package config

import (
 "os"
 "strings"
)

type TransparencyLog struct {
Enabled  bool   
Endpoint string 
}

type Features struct {
BiasDashboard bool 
ProviderMap   bool 
WsLiveUpdates bool 
}

type Constraints struct {
DefaultRegion string   // US|EU|ASIA
MaxCost       float64 
MaxDuration   int     
}

type Security struct {
RequireSignature     bool     
AllowedSubmitterKeys []string 
}

type Display struct {
MaintenanceMode bool   
Banner          string 
}

type AdminConfig struct {
IPFSGateway     string          
TransparencyLog TransparencyLog 
Features        Features        
Constraints     Constraints     
Security        Security        
Display         Display         
}

func DefaultAdminConfig() AdminConfig {
	gw := os.Getenv("IPFS_GATEWAY")
	if strings.TrimSpace(gw) == "" {
		gw = "https://ipfs.io"
	}
	te := os.Getenv("TRANSPARENCY_ENDPOINT")
	if strings.TrimSpace(te) == "" {
		te = "https://transparency.projectbeacon.dev"
	}
	return AdminConfig{
		IPFSGateway: gw,
		TransparencyLog: TransparencyLog{Enabled: true, Endpoint: te},
		Features: Features{BiasDashboard: true, ProviderMap: true, WsLiveUpdates: true},
		Constraints: Constraints{DefaultRegion: "US", MaxCost: 5.0, MaxDuration: 900},
		Security: Security{RequireSignature: true, AllowedSubmitterKeys: []string{}},
		Display: Display{MaintenanceMode: false, Banner: ""},
	}
}

type AdminConfigUpdate struct {
	IPFSGateway     *string `json:"ipfs_gateway"`
	TransparencyLog *struct {
		Enabled  *bool   `json:"enabled"`
		Endpoint *string `json:"endpoint"`
	} `json:"transparency_log"`
	Features *struct {
		BiasDashboard *bool `json:"bias_dashboard"`
		ProviderMap   *bool `json:"provider_map"`
		WsLiveUpdates *bool `json:"ws_live_updates"`
	} `json:"features"`
	Constraints *struct {
		DefaultRegion *string  `json:"default_region"`
		MaxCost       *float64 `json:"max_cost"`
		MaxDuration   *int     `json:"max_duration"`
	} `json:"constraints"`
	Security *struct {
		RequireSignature     *bool     `json:"require_signature"`
		AllowedSubmitterKeys *[]string `json:"allowed_submitter_keys"`
	} `json:"security"`
	Display *struct {
		MaintenanceMode *bool   `json:"maintenance_mode"`
		Banner          *string `json:"banner"`
	} `json:"display"`
}

func SanitizeAndMerge(cur AdminConfig, upd AdminConfigUpdate) AdminConfig {
	if upd.IPFSGateway != nil {
		cur.IPFSGateway = strings.TrimSpace(*upd.IPFSGateway)
	}
	if upd.TransparencyLog != nil {
		if upd.TransparencyLog.Enabled != nil { cur.TransparencyLog.Enabled = *upd.TransparencyLog.Enabled }
		if upd.TransparencyLog.Endpoint != nil { cur.TransparencyLog.Endpoint = strings.TrimSpace(*upd.TransparencyLog.Endpoint) }
	}
	if upd.Features != nil {
		if upd.Features.BiasDashboard != nil { cur.Features.BiasDashboard = *upd.Features.BiasDashboard }
		if upd.Features.ProviderMap != nil { cur.Features.ProviderMap = *upd.Features.ProviderMap }
		if upd.Features.WsLiveUpdates != nil { cur.Features.WsLiveUpdates = *upd.Features.WsLiveUpdates }
	}
	if upd.Constraints != nil {
		if upd.Constraints.DefaultRegion != nil {
			v := strings.ToUpper(strings.TrimSpace(*upd.Constraints.DefaultRegion))
			if v == "US" || v == "EU" || v == "ASIA" {
				cur.Constraints.DefaultRegion = v
			}
		}
		if upd.Constraints.MaxCost != nil { cur.Constraints.MaxCost = *upd.Constraints.MaxCost }
		if upd.Constraints.MaxDuration != nil { cur.Constraints.MaxDuration = *upd.Constraints.MaxDuration }
	}
	if upd.Security != nil {
		if upd.Security.RequireSignature != nil { cur.Security.RequireSignature = *upd.Security.RequireSignature }
		if upd.Security.AllowedSubmitterKeys != nil {
			var arr []string
			for _, s := range *upd.Security.AllowedSubmitterKeys {
				s = strings.TrimSpace(s)
				if s != "" { arr = append(arr, s) }
			}
			cur.Security.AllowedSubmitterKeys = arr
		}
	}
	if upd.Display != nil {
		if upd.Display.MaintenanceMode != nil { cur.Display.MaintenanceMode = *upd.Display.MaintenanceMode }
		if upd.Display.Banner != nil { cur.Display.Banner = *upd.Display.Banner }
	}
	return cur
}
