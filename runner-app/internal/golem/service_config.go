package golem

import (
	"context"
	"crypto/ed25519"
	"github.com/jamie-anson/project-beacon-runner/internal/golem/client"
)

// Network returns the configured network (e.g., mainnet, testnet)
func (s *Service) Network() string { return s.network }

// SigningKey returns the configured receipt signing private key (may be nil)
func (s *Service) SigningKey() ed25519.PrivateKey { return s.signingKey }

// Backend returns the active backend (mock or sdk)
func (s *Service) Backend() string { return s.backend }

// YagnaURL returns the configured Yagna base URL
func (s *Service) YagnaURL() string { return s.yagnaURL }

// AppKeyPresent indicates whether an app key is configured
func (s *Service) AppKeyPresent() bool { return s.yagnaKey != "" }

// ClientBases returns the current Market/Activity base prefixes if available
func (s *Service) ClientBases() (market string, activity string) {
	if yc, ok := s.client.(*client.YagnaRESTClient); ok {
		return yc.MarketBase, yc.ActivityBase
	}
	return "", ""
}

// ProbeOnce exposes a single probe attempt via the transport client
func (s *Service) ProbeOnce(ctx context.Context) (string, map[string]any, error) { return s.client.Probe(ctx) }
