package geoip

import (
	"testing"
)

func TestBucketFrom(t *testing.T) {
	cases := []struct{
		name string
		country string
		continent string
		want string
	}{
		{"us_explicit", "US", "NA", "US"},
		{"eu_continent", "DE", "EU", "EU"},
		{"asia_continent", "JP", "AS", "ASIA"},
		{"other", "BR", "SA", "OTHER"},
	}
	for _, tc := range cases {
		if got := bucketFromFields(tc.continent, tc.country); got != tc.want {
			t.Fatalf("%s: bucketFrom(country=%q, continent=%q)=%q want %q", tc.name, tc.country, tc.continent, got, tc.want)
		}
	}
}
