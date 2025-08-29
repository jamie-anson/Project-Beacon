package negotiation

import (
    "context"
    "fmt"
    "io"
    "net/http"
    "time"
)

// DefaultHTTPIPFetcher queries a public IP echo service to learn the caller's public IP.
// Note: In a real provider preflight, this should call a provider-exposed echo endpoint scoped by agreementID.
func DefaultHTTPIPFetcher(timeout time.Duration) ipFetcher {
    if timeout <= 0 {
        timeout = 5 * time.Second
    }
    client := &http.Client{Timeout: timeout}
    // Use a reliable plain-text endpoint.
    const ipURL = "https://api.ipify.org"

    return func(ctx context.Context, _ string) (string, error) {
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, ipURL, nil)
        if err != nil {
            return "", fmt.Errorf("build ip request: %w", err)
        }
        resp, err := client.Do(req)
        if err != nil {
            return "", fmt.Errorf("ip request failed: %w", err)
        }
        defer resp.Body.Close()
        if resp.StatusCode < 200 || resp.StatusCode >= 300 {
            return "", fmt.Errorf("ip service bad status: %s", resp.Status)
        }
        b, err := io.ReadAll(resp.Body)
        if err != nil {
            return "", fmt.Errorf("read ip response: %w", err)
        }
        return string(b), nil
    }
}
