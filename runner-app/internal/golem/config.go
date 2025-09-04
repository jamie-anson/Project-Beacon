package golem

import (
	"net/http"
	"os"
	"strings"
	"time"

	pcrypto "github.com/jamie-anson/project-beacon-runner/pkg/crypto"
)

// LoadConfigFromEnv constructs a Config using provided apiKey/network and environment variables.
func LoadConfigFromEnv(apiKey, network string) Config {
	cfg := Config{
		APIKey:       apiKey,
		Network:      network,
		Timeout:      30 * time.Minute,
		Backend:      "mock",
		YagnaURL:     "http://127.0.0.1:7465",
		YagnaKey:     os.Getenv("YAGNA_APPKEY"),
		HTTPClient:   &http.Client{Timeout: 15 * time.Second},
		EnableRealExec: os.Getenv("GOLEM_ENABLE_REAL_EXEC") == "true",
	}

	if b := os.Getenv("GOLEM_BACKEND"); b != "" {
		cfg.Backend = b
	}
	if url := os.Getenv("YAGNA_API_URL"); url != "" {
		cfg.YagnaURL = url
	}

	// Receipt signing key from env (base64). If absent, leave nil; caller may generate ephemeral.
	if keyB64 := os.Getenv("RECEIPT_PRIVATE_KEY"); keyB64 != "" {
		if pk, err := pcrypto.PrivateKeyFromBase64(keyB64); err == nil {
			cfg.SigningKey = pk
		}
	}

	// Optional overrides for Yagna base paths
	if mb := os.Getenv("GOLEM_MARKET_BASE"); mb != "" {
		if !strings.HasPrefix(mb, "/") { mb = "/" + mb }
		cfg.MarketBase = mb
	}
	if ab := os.Getenv("GOLEM_ACTIVITY_BASE"); ab != "" {
		if !strings.HasPrefix(ab, "/") { ab = "/" + ab }
		cfg.ActivityBase = ab
	}

	return cfg
}
