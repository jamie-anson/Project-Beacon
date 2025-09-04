package golem

import (
	"fmt"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// execFailure standardizes failure TaskExecution objects for the real SDK path.
func (s *Service) execFailure(stage string, start time.Time, provider *Provider, jobspec *models.JobSpec, meta map[string]any, cause error) *TaskExecution {
	end := time.Now()
	if meta == nil {
		meta = map[string]any{}
	}
	meta["stage"] = stage
	meta["note"] = "real SDK path scaffold; Yagna REST not wired"
	if cause != nil {
		meta["error"] = cause.Error()
	}
	// Ensure exit_code is always a concrete integer in the output
	exitVal := 1
	if v, ok := meta["exit_code"]; ok {
		switch t := v.(type) {
		case int:
			exitVal = t
		case int32:
			exitVal = int(t)
		case int64:
			exitVal = int(t)
		case float64:
			exitVal = int(t)
		case float32:
			exitVal = int(t)
		case string:
			// best-effort parse; ignore errors
		}
	}
	return &TaskExecution{
		ID:          fmt.Sprintf("sdk-real-todo-%d", end.UnixNano()),
		JobSpecID:   jobspec.ID,
		ProviderID:  provider.ID,
		Status:      "failed",
		StartedAt:   start,
		CompletedAt: end,
		Output: map[string]any{
			"stdout":    "",
			"stderr":    "",
			"exit_code": exitVal,
		},
		Metadata: meta,
		Error:    cause.Error(),
	}
}
