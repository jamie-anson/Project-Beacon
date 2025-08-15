package golem

import (
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// buildDemandSpec constructs the minimal DemandSpec from provider/jobspec constraints.
func (s *Service) buildDemandSpec(provider *Provider, jobspec *models.JobSpec) DemandSpec {
	c := map[string]any{
		"region": provider.Region,
		"image":  jobspec.Benchmark.Container.Image,
		"resources": map[string]any{
			"cpu":    jobspec.Benchmark.Container.Resources.CPU,
			"memory": jobspec.Benchmark.Container.Resources.Memory,
		},
	}
	return DemandSpec{
		Constraints: c,
		Metadata: map[string]any{
			"jobspec_id": jobspec.ID,
		},
	}
}
