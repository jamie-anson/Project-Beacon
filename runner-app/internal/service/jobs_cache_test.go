package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

type fakeCache struct{ data map[string][]byte }

func (f *fakeCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	b, ok := f.data[key]
	return b, ok, nil
}
func (f *fakeCache) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	if f.data == nil { f.data = make(map[string][]byte) }
	f.data[key] = val
	return nil
}

func TestGetJob_CacheHit(t *testing.T) {
	s := &JobsService{}
	fc := &fakeCache{data: map[string][]byte{}}
	s.SetCache(fc)

	payload, _ := json.Marshal(struct{
		Spec   models.JobSpec `json:"spec"`
		Status string         `json:"status"`
	}{Spec: models.JobSpec{ID: "job-abc", Version: "1.0"}, Status: "running"})
	fc.data["job:job-abc"] = payload

	spec, status, err := s.GetJob(context.Background(), "job-abc")
	if err != nil { t.Fatalf("err: %v", err) }
	if spec == nil || spec.ID != "job-abc" || status != "running" {
		t.Fatalf("unexpected: spec=%#v status=%s", spec, status)
	}
}

func TestGetLatestReceiptCached_CacheHit(t *testing.T) {
	s := &JobsService{}
	fc := &fakeCache{data: map[string][]byte{}}
	s.SetCache(fc)

	rec := &models.Receipt{ExecutionDetails: models.ExecutionDetails{Status: "completed"}}
	b, _ := json.Marshal(rec)
	fc.data["job:job-xyz:latest_receipt"] = b

	got, err := s.GetLatestReceiptCached(context.Background(), "job-xyz")
	if err != nil { t.Fatalf("err: %v", err) }
	if got == nil || got.ExecutionDetails.Status != "completed" {
		t.Fatalf("unexpected receipt: %#v", got)
	}
}
