package api

import (
    "encoding/json"
    "os"
    "path/filepath"
    "reflect"
    "testing"
)

func mustReadGolden(t *testing.T, name string) []byte {
    t.Helper()
    p := filepath.Join("testdata", name)
    b, err := os.ReadFile(p)
    if err != nil {
        t.Fatalf("read golden %s: %v", p, err)
    }
    return b
}

func jsonEqual(t *testing.T, got []byte, golden []byte) {
    t.Helper()
    var g1 any
    var g2 any
    if err := json.Unmarshal(got, &g1); err != nil { t.Fatalf("unmarshal got: %v; body=%s", err, string(got)) }
    if err := json.Unmarshal(golden, &g2); err != nil { t.Fatalf("unmarshal golden: %v; body=%s", err, string(golden)) }
    if !reflect.DeepEqual(g1, g2) {
        t.Fatalf("json mismatch\nGOT:   %s\nWANT:  %s", string(got), string(golden))
    }
}
