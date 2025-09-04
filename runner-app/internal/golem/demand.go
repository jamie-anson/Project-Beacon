package golem

import (
	"fmt"
	clientpkg "github.com/jamie-anson/project-beacon-runner/internal/golem/client"
	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

// buildDemandSpec constructs the minimal DemandSpec from provider/jobspec constraints.
func (s *Service) buildDemandSpec(provider *Provider, jobspec *models.JobSpec) clientpkg.DemandSpec {
	// Yagna expects constraints as a string expression, not a map
	constraintsExpr := fmt.Sprintf("(&(golem.runtime.name=vm)(golem.runtime.vm.image.hash=%s))", 
		jobspec.Benchmark.Container.Image)
	
	return clientpkg.DemandSpec{
		Constraints: constraintsExpr,
		Properties: map[string]any{
			"golem.runtime.name": "vm",
			"golem.runtime.vm.image.hash": jobspec.Benchmark.Container.Image,
		},
		Metadata: map[string]any{
			"jobspec_id": jobspec.ID,
			"region":     provider.Region,
		},
	}
}
