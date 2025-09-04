package models

import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "reflect"
    "testing"
    "time"
)

func validJobSpec() *JobSpec {
    return &JobSpec{
        ID:      "job_123",
        Version: "v1",
        Benchmark: BenchmarkSpec{
            Name: "bench",
            Container: ContainerSpec{
                Image: "repo/image",
                Tag:   "latest",
                Resources: ResourceSpec{
                    CPU:    "1000m",
                    Memory: "512Mi",
                },
            },
            Input: InputSpec{
                Type: "prompt",
                Data: map[string]any{"q": "hello"},
                Hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", // sha256("")
            },
            Scoring:  ScoringSpec{Method: "custom", Parameters: map[string]any{"k": 1}},
            Metadata: map[string]string{"m": "v"},
        },
        Constraints: ExecutionConstraints{
            Regions:         []string{"us", "eu", "apac"},
            MinRegions:      2,
            MinSuccessRate:  0.5,
            Timeout:         5 * time.Minute,
            ProviderTimeout: 1 * time.Minute,
        },
        Metadata:  map[string]any{"owner": "alice"},
        CreatedAt: time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
        Signature: "",
        PublicKey: "",
    }
}

func TestValidateRegions_EdgeCases(t *testing.T) {
    v := NewJobSpecValidator()
    // Mixed valid set meeting min
    if err := v.ValidateRegions([]string{"US", "EU", "APAC"}, 2); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    // Lowercase regions should fail because supported set is uppercase-only
    if err := v.ValidateRegions([]string{"us", "eu"}, 1); err == nil {
        t.Fatalf("expected unsupported region error for lowercase regions")
    }
    // Include one invalid among many
    if err := v.ValidateRegions([]string{"US", "EU", "MIDDLE-EARTH"}, 2); err == nil {
        t.Fatalf("expected unsupported region error for MIDDLE-EARTH")
    }
}

func TestValidateContainerImage_EdgeCases(t *testing.T) {
    v := NewJobSpecValidator()
    // Valid complex registry paths
    casesValid := [][2]string{
        {"ghcr.io/org/proj", "v1.2.3"},
        {"docker.io/library/nginx", "1.25"},
        {"repo.subdomain-1/path_name.more", "rc-1"},
    }
    for _, c := range casesValid {
        if err := v.ValidateContainerImage(c[0], c[1]); err != nil {
            t.Fatalf("unexpected error for valid image %s:%s -> %v", c[0], c[1], err)
        }
    }

    // Invalid images: uppercase, leading slash, invalid chars
    casesInvalidImage := []string{
        "/leading/slash",
        "Upper/Case/Image",
        "repo/space in name",
        "repo/invalid!char",
    }
    for _, img := range casesInvalidImage {
        if err := v.ValidateContainerImage(img, "latest"); err == nil {
            t.Fatalf("expected error for invalid image %q", img)
        }
    }

    // Invalid tags: illegal chars, spaces, slash
    casesInvalidTag := []string{"@bad", "bad tag", "slash/inside"}
    for _, tag := range casesInvalidTag {
        if err := v.ValidateContainerImage("repo/image", tag); err == nil {
            t.Fatalf("expected error for invalid tag %q", tag)
        }
    }
}

func TestValidateJobSpecID(t *testing.T) {
    v := NewJobSpecValidator()
    if err := v.ValidateJobSpecID("ok_ID-123"); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if err := v.ValidateJobSpecID(""); err == nil {
        t.Fatalf("expected error for empty id")
    }
    if err := v.ValidateJobSpecID("bad id!"); err == nil {
        t.Fatalf("expected invalid character error")
    }
}

func TestValidateContainerImage(t *testing.T) {
    v := NewJobSpecValidator()
    if err := v.ValidateContainerImage("repo/image", "latest"); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if err := v.ValidateContainerImage("", ""); err == nil {
        t.Fatalf("expected error for empty image")
    }
    if err := v.ValidateContainerImage("INVALID!/name", ""); err == nil {
        t.Fatalf("expected invalid image format error")
    }
    if err := v.ValidateContainerImage("repo/image", "bad tag!* "); err == nil {
        t.Fatalf("expected invalid tag format error")
    }
}

func TestValidateRegions(t *testing.T) {
    v := NewJobSpecValidator()
    if err := v.ValidateRegions([]string{"US", "EU"}, 2); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if err := v.ValidateRegions([]string{"US"}, 2); err == nil {
        t.Fatalf("expected insufficient regions error")
    }
    if err := v.ValidateRegions([]string{"US", "MARS"}, 1); err == nil {
        t.Fatalf("expected unsupported region error")
    }
}

func TestValidateTimeout(t *testing.T) {
    v := NewJobSpecValidator()
    if err := v.ValidateTimeout(45 * time.Second); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if err := v.ValidateTimeout(1 * time.Second); err == nil {
        t.Fatalf("expected too short error")
    }
    if err := v.ValidateTimeout(2 * time.Hour); err == nil {
        t.Fatalf("expected too long error")
    }
}

func TestValidateInputHash(t *testing.T) {
    v := NewJobSpecValidator()
    // valid plain hex
    if err := v.ValidateInputHash("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    // valid with sha256: prefix
    if err := v.ValidateInputHash("sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"); err != nil {
        t.Fatalf("unexpected error with prefix: %v", err)
    }
    if err := v.ValidateInputHash(""); err == nil {
        t.Fatalf("expected error for empty hash")
    }
    if err := v.ValidateInputHash("zzz"); err == nil {
        t.Fatalf("expected invalid length error")
    }
    if err := v.ValidateInputHash("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaag"); err == nil { // last char not hex
        t.Fatalf("expected invalid hex error")
    }
}

func TestComputeJobSpecHash_DeterministicAndExcludesSignature(t *testing.T) {
    v := NewJobSpecValidator()
    js := validJobSpec()
    // baseline
    h1, err := v.ComputeJobSpecHash(js)
    if err != nil {
        t.Fatalf("hash error: %v", err)
    }
    // same content -> same hash
    h2, _ := v.ComputeJobSpecHash(js)
    if h1 != h2 {
        t.Fatalf("expected deterministic hash, got %s vs %s", h1, h2)
    }
    // add signature/public key -> hash must remain the same
    js.Signature = "AAA"
    js.PublicKey = "BBB"
    h3, _ := v.ComputeJobSpecHash(js)
    if h1 != h3 {
        t.Fatalf("expected same hash when only signature/public key changes")
    }
    // mutate field -> hash changes
    js.Benchmark.Name = "bench2"
    h4, _ := v.ComputeJobSpecHash(js)
    if h1 == h4 {
        t.Fatalf("expected hash change after content mutation")
    }

    // sanity check with local sha256 of marshaled copy per implementation
    js2 := *js
    js2.Signature = ""
    js2.PublicKey = ""
    b := mustJSON(js2)
    want := sha256.Sum256(b)
    if hex.EncodeToString(want[:]) != h4 {
        t.Fatalf("expected implementation-compatible hash")
    }
}

func TestSanitizeJobSpec_NormalizesAndDefaults(t *testing.T) {
    v := NewJobSpecValidator()
    js := validJobSpec()
    // lower-case regions and zero defaults
    js.Constraints.Regions = []string{"us", "eu"}
    js.Constraints.MinRegions = 0
    js.Constraints.Timeout = 0
    js.Metadata = nil
    js.Benchmark.Metadata = nil

    v.SanitizeJobSpec(js)

    // Regions uppercased
    if !reflect.DeepEqual(js.Constraints.Regions, []string{"US", "EU"}) {
        t.Fatalf("regions not uppercased: %+v", js.Constraints.Regions)
    }
    // MinRegions defaulted to len(regions)
    if js.Constraints.MinRegions != 2 {
        t.Fatalf("expected MinRegions=2, got %d", js.Constraints.MinRegions)
    }
    // Timeout defaulted
    if js.Constraints.Timeout <= 0 {
        t.Fatalf("expected non-zero timeout default")
    }
    if js.Metadata == nil || js.Benchmark.Metadata == nil {
        t.Fatalf("expected metadata maps initialized")
    }
}

func TestExtractJobSpecSummary(t *testing.T) {
    v := NewJobSpecValidator()
    js := validJobSpec()
    sum := v.ExtractJobSpecSummary(js)
    if sum["id"] != js.ID || sum["version"] != js.Version {
        t.Fatalf("id/version mismatch")
    }
    if sum["container"].(string) != js.Benchmark.Container.Image+":"+js.Benchmark.Container.Tag {
        t.Fatalf("container mismatch")
    }
    if sum["has_signature"].(bool) != (js.Signature != "") {
        t.Fatalf("has_signature mismatch")
    }
    if _, ok := sum["created_at"].(string); !ok {
        t.Fatalf("created_at should be string")
    }
}

// helper to marshal deterministically
func mustJSON(v any) []byte {
    b, err := jsonMarshal(v)
    if err != nil {
        panic(err)
    }
    return b
}

// small indirection to avoid import collision with models' json tags
func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
