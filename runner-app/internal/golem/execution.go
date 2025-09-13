package golem

import (
    "context"
    "fmt"
    "time"

    "github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// ExecuteTask runs a benchmark task on a specific provider
func (s *Service) ExecuteTask(ctx context.Context, provider *Provider, jobspec *models.JobSpec) (*TaskExecution, error) {
    // If using SDK backend, delegate to SDK implementation
    if s.backend == "sdk" {
        return s.executeTaskSDK(ctx, provider, jobspec)
    }

    execution := &TaskExecution{
        ID:         fmt.Sprintf("task_%d", time.Now().Unix()),
        JobSpecID:  jobspec.ID,
        ProviderID: provider.ID,
        Status:     "pending",
        StartedAt:  time.Now(),
        Metadata:   make(map[string]interface{}),
    }

    // Add execution metadata
    execution.Metadata["provider_region"] = provider.Region
    execution.Metadata["provider_score"] = provider.Score
    execution.Metadata["benchmark_name"] = jobspec.Benchmark.Name

    // For MVP, simulate task execution
    if err := s.simulateTaskExecution(ctx, execution, jobspec); err != nil {
        execution.Status = "failed"
        execution.Error = err.Error()
        execution.CompletedAt = time.Now()
        return execution, err
    }

    execution.Status = "completed"
    execution.CompletedAt = time.Now()

    return execution, nil
}

// simulateTaskExecution simulates running a benchmark task
func (s *Service) simulateTaskExecution(ctx context.Context, execution *TaskExecution, jobspec *models.JobSpec) error {
    // Simulate execution time based on benchmark complexity
    executionTime := 5 * time.Second // Base execution time

    // Add some randomness to simulate real execution
    if jobspec.Benchmark.Name == "Who Are You?" {
        executionTime = 3 * time.Second
    }

    select {
    case <-time.After(executionTime):
        // Simulate successful execution
        execution.Status = "running"

        // Generate structured output based on region and questions
        output := s.generateStructuredOutput(execution.ProviderID, jobspec)
        execution.Output = output

        // Populate process-like outputs (stdout/stderr/exit)
        // Extract summary from structured output for stdout
        if m, ok := output.(map[string]interface{}); ok {
            if data, ok := m["data"].(map[string]interface{}); ok {
                if summary, ok := data["summary"].(map[string]interface{}); ok {
                    stdout := fmt.Sprintf("Processed %v questions, %v successful, %v failed, total time: %.2fs", 
                        summary["total_questions"], 
                        summary["successful_responses"], 
                        summary["failed_responses"],
                        summary["total_inference_time"])
                    execution.Metadata["stdout"] = stdout
                }
            }
        }
        execution.Metadata["stderr"] = ""
        execution.Metadata["exit_code"] = 0

        return nil

    case <-ctx.Done():
        return fmt.Errorf("task execution cancelled: %w", ctx.Err())
    }
}
