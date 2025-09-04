package transparency

import (
	"context"

	"github.com/jamie-anson/project-beacon-runner/internal/ipfs"
)

// IPFSStorage is a thin wrapper around internal/ipfs.Client for transparency bundles.
type IPFSStorage struct {
	client *ipfs.Client
}

// NewIPFSStorage creates storage with provided client; if nil, builds from env.
func NewIPFSStorage(client *ipfs.Client) *IPFSStorage {
	if client == nil {
		client = ipfs.NewFromEnv()
	}
	return &IPFSStorage{client: client}
}

// StoreBundle adds and pins the given bundle, returning its CID and gateway URL.
func (s *IPFSStorage) StoreBundle(ctx context.Context, bundle *ipfs.Bundle) (cid string, gatewayURL string, err error) {
	cid, err = s.client.AddAndPin(ctx, bundle)
	if err != nil {
		return "", "", err
	}
	return cid, s.client.GetGatewayURL(cid), nil
}

// FetchBundle retrieves a bundle by CID.
func (s *IPFSStorage) FetchBundle(ctx context.Context, cid string) (*ipfs.Bundle, error) {
	return s.client.GetBundle(ctx, cid)
}

// PinCID ensures a CID remains pinned.
func (s *IPFSStorage) PinCID(ctx context.Context, cid string) error {
	return s.client.PinBundle(ctx, cid)
}

// Connected reports whether the underlying node is reachable.
func (s *IPFSStorage) Connected(ctx context.Context) bool {
	return s.client.IsConnected(ctx)
}
