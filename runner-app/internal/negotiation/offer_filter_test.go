package negotiation

import "testing"

func TestOfferFilter_Classify(t *testing.T) {
	f := NewOfferFilter()

	cases := []struct{
		name string
		offer OfferView
		target string
		want MatchLevel
		claimed string
	}{
		{"p0_beacon_region", OfferView{Properties: map[string]string{"beacon.region":"US"}}, "US", P0Explicit, "US"},
		{"p1_region", OfferView{Properties: map[string]string{"region":"EU"}}, "EU", P1Generic, "EU"},
		{"p1_geo_region", OfferView{Properties: map[string]string{"geo.region":"EUROPE"}}, "EU", P1Generic, "EUROPE"},
		{"p2_tags", OfferView{Tags: []string{"gpu","asia"}}, "ASIA", P2Tags, "asia"},
		{"p3_needs_probe", OfferView{Properties: map[string]string{"foo":"bar"}}, "US", P3NeedsProbe, ""},
		{"empty_target", OfferView{}, "", P3NeedsProbe, ""},
	}
	for _, tc := range cases {
		lvl, claimed := f.Classify(tc.offer, tc.target)
		if lvl != tc.want {
			t.Fatalf("%s: got level %v want %v", tc.name, lvl, tc.want)
		}
		if claimed != tc.claimed {
			t.Fatalf("%s: claimed %q want %q", tc.name, claimed, tc.claimed)
		}
	}
}

func TestNormalizeRegion(t *testing.T) {
	cases := map[string]string{
		"us": "US",
		"USA": "US",
		"United States": "US",
		"EU": "EU",
		"Europe": "EU",
		"EUROPEAN UNION": "EU",
		"apac": "ASIA",
		"AS": "ASIA",
		"Asian": "ASIA",
		"unknown": "",
	}
	for in, want := range cases {
		if got := normalizeRegion(in); got != want {
			t.Fatalf("normalize(%q)=%q want %q", in, got, want)
		}
	}
}
