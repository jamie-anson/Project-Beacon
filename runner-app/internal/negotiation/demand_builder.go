package negotiation

// DemandBuilder builds market demands with hard constraints.
type DemandBuilder interface {
    Build(req RegionRequest) (DemandSpec, error)
}

type demandBuilder struct{}

func NewDemandBuilder() DemandBuilder { return &demandBuilder{} }

func (d *demandBuilder) Build(req RegionRequest) (DemandSpec, error) {
    // Construct a DemandSpec enforcing hard constraints; region used as soft filter.
    spec := DemandSpec{
        Runtime:           "docker",
        MinVCPU:           req.MinVCPU,
        MinMemGiB:         req.MinMemGiB,
        NetworkEgress:     req.NetworkEgress,
        PricePerMinuteMax: req.PricePerMinuteMax,
        TotalPriceCap:     req.TotalPriceCap,
        Region:            req.Region,
    }
    return spec, nil
}
