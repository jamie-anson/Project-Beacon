package metrics

import (
    "context"
    "errors"
    "testing"

    "github.com/jamie-anson/project-beacon-runner/internal/store"
    "github.com/prometheus/client_golang/prometheus/testutil"
    "github.com/prometheus/client_golang/prometheus"
)

// fakes implementing the small interfaces
type fakeIPFSRepo struct {
    bundles []store.IPFSBundle
    err     error
}

func (f *fakeIPFSRepo) ListBundles(limit, offset int) ([]store.IPFSBundle, error) {
    if f.err != nil {
        return nil, f.err
    }
    return f.bundles, nil
}

type fakeTransRepo struct {
    size     int64
    sizeErr  error
    valid    bool
    validErr error
}

func (f *fakeTransRepo) GetLogSize() (int64, error) {
    if f.sizeErr != nil {
        return 0, f.sizeErr
    }
    return f.size, nil
}

func (f *fakeTransRepo) VerifyLogIntegrity() (bool, error) {
    if f.validErr != nil {
        return false, f.validErr
    }
    return f.valid, nil
}

func TestUpdateStorageMetrics_Error(t *testing.T) {
    resetProm()
    c := NewCollector(nil, nil)
    c.ipfsRepo = &fakeIPFSRepo{err: errors.New("boom")}

    if err := c.UpdateStorageMetrics(ctxBg()); err == nil {
        t.Fatalf("expected error, got nil")
    }
}

func TestUpdateStorageMetrics_Success_SetsGauges(t *testing.T) {
    resetProm()
    c := NewCollector(nil, nil)
    size1, size2 := int64(100), int64(300)
    bundles := []store.IPFSBundle{{BundleSize: &size1}, {BundleSize: &size2}, {BundleSize: nil}}
    c.ipfsRepo = &fakeIPFSRepo{bundles: bundles}

    if err := c.UpdateStorageMetrics(ctxBg()); err != nil {
        t.Fatalf("unexpected err: %v", err)
    }

    got := testutil.ToFloat64(c.ipfsStorageTotal)
    want := float64(size1 + size2)
    if got != want {
        t.Fatalf("ipfsStorageTotal = %v, want %v", got, want)
    }

    // storageEfficiency is a fixed placeholder 0.75
    eff := testutil.ToFloat64(c.storageEfficiency)
    if eff != 0.75 {
        t.Fatalf("storageEfficiency = %v, want 0.75", eff)
    }
}

func TestUpdateTransparencyMetrics_SizeError(t *testing.T) {
    resetProm()
    c := NewCollector(nil, nil)
    c.transparencyRepo = &fakeTransRepo{sizeErr: errors.New("db down")}

    if err := c.UpdateTransparencyMetrics(ctxBg()); err == nil {
        t.Fatalf("expected error, got nil")
    }
}

func TestUpdateTransparencyMetrics_VerifyError(t *testing.T) {
    resetProm()
    c := NewCollector(nil, nil)
    c.transparencyRepo = &fakeTransRepo{size: 10, validErr: errors.New("verify failed")}

    if err := c.UpdateTransparencyMetrics(ctxBg()); err == nil {
        t.Fatalf("expected error, got nil")
    }
}

func TestUpdateTransparencyMetrics_SetsGauges(t *testing.T) {
    resetProm()
    c := NewCollector(nil, nil)
    c.transparencyRepo = &fakeTransRepo{size: 5, valid: true}

    if err := c.UpdateTransparencyMetrics(ctxBg()); err != nil {
        t.Fatalf("unexpected err: %v", err)
    }

    if got := testutil.ToFloat64(c.transparencyLogEntries); got != 5 {
        t.Fatalf("transparencyLogEntries = %v, want 5", got)
    }
    if got := testutil.ToFloat64(c.transparencyLogIntegrity); got != 1 {
        t.Fatalf("transparencyLogIntegrity = %v, want 1", got)
    }

    // Now set invalid path
    c.transparencyRepo = &fakeTransRepo{size: 7, valid: false}
    if err := c.UpdateTransparencyMetrics(ctxBg()); err != nil {
        t.Fatalf("unexpected err: %v", err)
    }
    if got := testutil.ToFloat64(c.transparencyLogEntries); got != 7 {
        t.Fatalf("transparencyLogEntries = %v, want 7", got)
    }
    if got := testutil.ToFloat64(c.transparencyLogIntegrity); got != 0 {
        t.Fatalf("transparencyLogIntegrity = %v, want 0", got)
    }
}

// helper: background context without importing context in each test
func ctxBg() context.Context { return context.Background() }

// resetProm resets the default prometheus registry to avoid duplicate registrations across tests
func resetProm() {
    reg := prometheus.NewRegistry()
    prometheus.DefaultRegisterer = reg
    prometheus.DefaultGatherer = reg
    // Re-register all package metrics via helper
    RegisterAll()
}
