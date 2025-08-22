package anchor

import (
    "context"
    "testing"
    "time"
)

func TestTimestampStrategy_AnchorVerifyStatus(t *testing.T) {
    ts := NewTimestampStrategy("")
    if ts.TSAUrl == "" { t.Fatalf("expected default TSAUrl") }

    ctx := context.Background()
    res, err := ts.Anchor(ctx, "abc123")
    if err != nil { t.Fatalf("Anchor err: %v", err) }
    if res == nil || res.Network != "timestamp" || res.AnchoredHash != "abc123" {
        t.Fatalf("unexpected result: %#v", res)
    }

    ok, err := ts.Verify(ctx, res)
    if err != nil || !ok { t.Fatalf("Verify not ok: %v %v", ok, err) }

    st, err := ts.GetStatus(ctx, res.TxHash)
    if err != nil { t.Fatalf("GetStatus err: %v", err) }
    if st.Status != "confirmed" || st.Confirmations < 1 {
        t.Fatalf("unexpected status: %#v", st)
    }
}

func TestEthereumStrategy_AnchorVerifyStatus(t *testing.T) {
    es := NewEthereumStrategy("http://rpc", "pk", "0xdead")
    ctx := context.Background()

    res, err := es.Anchor(ctx, "deadbeef")
    if err != nil { t.Fatalf("Anchor err: %v", err) }
    if res.Network != "ethereum" || res.Cost == "" || res.BlockNumber == 0 {
        t.Fatalf("unexpected res: %#v", res)
    }

    ok, err := es.Verify(ctx, res)
    if err != nil || !ok { t.Fatalf("Verify err: %v ok=%v", err, ok) }

    st, err := es.GetStatus(ctx, res.TxHash)
    if err != nil { t.Fatalf("GetStatus err: %v", err) }
    if st.Status != "confirmed" || st.Confirmations < 1 {
        t.Fatalf("unexpected status: %#v", st)
    }
}

// mock strategy to force failures
type failingStrategy struct{}
func (f failingStrategy) Anchor(ctx context.Context, hash string) (*AnchorResult, error) { return nil, context.Canceled }
func (f failingStrategy) Verify(ctx context.Context, result *AnchorResult) (bool, error) { return false, context.Canceled }
func (f failingStrategy) GetStatus(ctx context.Context, txHash string) (*AnchorStatus, error) { return nil, context.Canceled }

// mock strategy to force deterministic success
type staticStrategy struct{ res *AnchorResult }
func (s staticStrategy) Anchor(ctx context.Context, hash string) (*AnchorResult, error) { return s.res, nil }
func (s staticStrategy) Verify(ctx context.Context, result *AnchorResult) (bool, error) { return result.TxHash == s.res.TxHash, nil }
func (s staticStrategy) GetStatus(ctx context.Context, txHash string) (*AnchorStatus, error) { return &AnchorStatus{TxHash: txHash, Status: "confirmed", Confirmations: 1, Timestamp: time.Now()}, nil }

func TestMultiStrategy_SuccessAndFailure(t *testing.T) {
    ts := time.Now()
    good := staticStrategy{res: &AnchorResult{TxHash: "xyz", AnchoredHash: "h1", Timestamp: ts, Network: "test"}}
    ms := NewMultiStrategy(failingStrategy{}, good)

    res, err := ms.Anchor(context.Background(), "h1")
    if err != nil || res == nil { t.Fatalf("expected success: %v %#v", err, res) }

    ok, err := ms.Verify(context.Background(), res)
    if err != nil || !ok { t.Fatalf("verify ok: %v %v", ok, err) }

    st, err := ms.GetStatus(context.Background(), res.TxHash)
    if err != nil || st == nil { t.Fatalf("status err: %v", err) }
}

func TestMultiStrategy_AllFail(t *testing.T) {
    ms := NewMultiStrategy(failingStrategy{}, failingStrategy{})
    if _, err := ms.Anchor(context.Background(), "h" ); err == nil {
        t.Fatalf("expected error when all fail")
    }
    if ok, _ := ms.Verify(context.Background(), &AnchorResult{TxHash:"nope"}); ok {
        t.Fatalf("expected verify false")
    }
    if st, err := ms.GetStatus(context.Background(), "x"); err == nil || st != nil {
        t.Fatalf("expected status not found error")
    }
}
