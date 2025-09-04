package ipfs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"net/http"
	"time"

	shell "github.com/ipfs/go-ipfs-api"
	// OpenTelemetry
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Client wraps IPFS operations for Project Beacon
type Client struct {
	shell   *shell.Shell
	gateway string
}

// Config holds IPFS client configuration
type Config struct {
	NodeURL string
	Gateway string
}

// Bundle represents a complete execution bundle for IPFS storage
type Bundle struct {
	JobID       string                 `json:"job_id"`
	ExecutionID string                 `json:"execution_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Receipts    []Receipt              `json:"receipts"`
	Outputs     map[string]string      `json:"outputs"`     // region -> output
	Metadata    map[string]interface{} `json:"metadata"`
	Signature   string                 `json:"signature"`
	PublicKey   string                 `json:"public_key"`
}

// Receipt represents a cryptographic receipt for IPFS bundling
type Receipt struct {
	ExecutionID string    `json:"execution_id"`
	JobID       string    `json:"job_id"`
	Region      string    `json:"region"`
	ProviderID  string    `json:"provider_id"`
	Output      string    `json:"output"`
	OutputHash  string    `json:"output_hash"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Signature   string    `json:"signature"`
	PublicKey   string    `json:"public_key"`
}

// NewClient creates a new IPFS client
func NewClient(config Config) *Client {
	if config.NodeURL == "" {
		config.NodeURL = "localhost:5001" // Default IPFS API port
	}
	if config.Gateway == "" {
		config.Gateway = "http://localhost:8080" // Default IPFS gateway
	}

	return &Client{
		shell:   shell.NewShell(config.NodeURL),
		gateway: config.Gateway,
	}
}

// NewFromEnv creates an IPFS client from environment variables
func NewFromEnv() *Client {
	nodeURL := os.Getenv("IPFS_NODE_URL")
	gateway := os.Getenv("IPFS_GATEWAY")
	
	return NewClient(Config{
		NodeURL: nodeURL,
		Gateway: gateway,
	})
}

// AddBundle uploads a bundle to IPFS and returns the CID
func (c *Client) AddBundle(ctx context.Context, bundle *Bundle) (string, error) {
	tracer := otel.Tracer("runner/ipfs")
	ctx, span := tracer.Start(ctx, "IPFS.AddBundle", oteltrace.WithAttributes(
		attribute.String("job.id", bundle.JobID),
		attribute.String("execution.id", bundle.ExecutionID),
	))
	defer span.End()
	// Serialize bundle to JSON
	bundleData, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal bundle: %w", err)
	}

	// Add to IPFS
	reader := bytes.NewReader(bundleData)
	cid, err := c.shell.Add(reader)
	if err != nil {
		return "", fmt.Errorf("failed to add bundle to IPFS: %w", err)
	}

	return cid, nil
}

// PinBundle pins a bundle to ensure it stays in IPFS
func (c *Client) PinBundle(ctx context.Context, cid string) error {
	tracer := otel.Tracer("runner/ipfs")
	ctx, span := tracer.Start(ctx, "IPFS.PinBundle", oteltrace.WithAttributes(
		attribute.String("ipfs.cid", cid),
	))
	defer span.End()
	err := c.shell.Pin(cid)
	if err != nil {
		return fmt.Errorf("failed to pin bundle %s: %w", cid, err)
	}
	return nil
}

// GetBundle retrieves a bundle from IPFS by CID
func (c *Client) GetBundle(ctx context.Context, cid string) (*Bundle, error) {
	tracer := otel.Tracer("runner/ipfs")
	ctx, span := tracer.Start(ctx, "IPFS.GetBundle", oteltrace.WithAttributes(
		attribute.String("ipfs.cid", cid),
	))
	defer span.End()
	reader, err := c.shell.Cat(cid)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve bundle %s: %w", cid, err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read bundle data: %w", err)
	}

	var bundle Bundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bundle: %w", err)
	}

	return &bundle, nil
}

// AddAndPin combines adding and pinning a bundle in one operation
func (c *Client) AddAndPin(ctx context.Context, bundle *Bundle) (string, error) {
	tracer := otel.Tracer("runner/ipfs")
	ctx, span := tracer.Start(ctx, "IPFS.AddAndPin", oteltrace.WithAttributes(
		attribute.String("job.id", bundle.JobID),
		attribute.String("execution.id", bundle.ExecutionID),
	))
	defer span.End()
	cid, err := c.AddBundle(ctx, bundle)
	if err != nil {
		return "", err
	}

	if err := c.PinBundle(ctx, cid); err != nil {
		return cid, fmt.Errorf("bundle added but pinning failed: %w", err)
	}

	// Optionally also pin on Storacha if configured via environment variables.
	// Best-effort: if Storacha pin fails, we do NOT fail the flow; record a span event and still return the CID.
	if perr := pinWithStoracha(ctx, cid); perr != nil {
		span.AddEvent("storacha_pin_failed", oteltrace.WithAttributes(
			attribute.String("ipfs.cid", cid),
			attribute.String("error", perr.Error()),
		))
	}

	return cid, nil
}

// GetGatewayURL returns the IPFS gateway URL for a given CID
func (c *Client) GetGatewayURL(cid string) string {
	return fmt.Sprintf("%s/ipfs/%s", c.gateway, cid)
}

// pinWithStoracha posts the CID to an external Storacha pin endpoint if configured.
// Environment variables:
//   STORACHA_PIN_URL - full URL to the pin-by-CID endpoint
//   STORACHA_TOKEN   - bearer token for authorization (optional if endpoint is public)
// Behavior:
//   - If STORACHA_PIN_URL is empty, this is a no-op and returns nil.
//   - Sends POST with JSON body: {"cid": "<cid>"} and Content-Type: application/json
//   - Adds Authorization: Bearer <token> header when STORACHA_TOKEN is set
func pinWithStoracha(ctx context.Context, cid string) error {
    pinURL := os.Getenv("STORACHA_PIN_URL")
    if pinURL == "" {
        return nil
    }
    payload := map[string]string{"cid": cid}
    body, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("storacha: marshal payload: %w", err)
    }
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, pinURL, bytes.NewReader(body))
    if err != nil {
        return fmt.Errorf("storacha: build request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    if tok := os.Getenv("STORACHA_TOKEN"); tok != "" {
        req.Header.Set("Authorization", "Bearer "+tok)
    }

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return fmt.Errorf("storacha: request failed: %w", err)
    }
    defer resp.Body.Close()
    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("storacha: unexpected status %d: %s", resp.StatusCode, string(b))
    }
    return nil
}

// IsConnected checks if the IPFS node is reachable
func (c *Client) IsConnected(ctx context.Context) bool {
	tracer := otel.Tracer("runner/ipfs")
	ctx, span := tracer.Start(ctx, "IPFS.IsConnected")
	defer span.End()
	_, err := c.shell.ID()
	return err == nil
}

// NodeInfo returns information about the connected IPFS node
func (c *Client) NodeInfo(ctx context.Context) (map[string]interface{}, error) {
	tracer := otel.Tracer("runner/ipfs")
	ctx, span := tracer.Start(ctx, "IPFS.NodeInfo")
	defer span.End()
	info, err := c.shell.ID()
	if err != nil {
		return nil, fmt.Errorf("failed to get node info: %w", err)
	}

	return map[string]interface{}{
		"id":              info.ID,
		"public_key":      info.PublicKey,
		"addresses":       info.Addresses,
		"agent_version":   info.AgentVersion,
		"protocol_version": info.ProtocolVersion,
	}, nil
}
