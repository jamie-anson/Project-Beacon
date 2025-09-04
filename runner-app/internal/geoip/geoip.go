package geoip

import (
    "errors"
    "fmt"
    "net"
    "os"
    "sync"

    "github.com/oschwald/geoip2-golang"
)

// Resolver performs IP -> country/region lookups.
type Resolver interface {
    // LookupIP returns country ISO code, region bucket, source label, and error.
    // regionBucket is one of: "US", "EU", "ASIA", or "OTHER" when unknown.
    LookupIP(ip string) (country string, regionBucket string, source string, err error)
}

type resolver struct {
    once sync.Once
    db   *geoip2.Reader
    err  error
}

// NewResolver constructs a GeoLite2-backed resolver. It defers DB opening until first use.
// Set GEOIP_DB_PATH to the path of GeoLite2-City.mmdb.
func NewResolver() Resolver { return &resolver{} }

func (r *resolver) open() error {
    r.once.Do(func() {
        path := os.Getenv("GEOIP_DB_PATH")
        if path == "" {
            r.err = errors.New("GEOIP_DB_PATH not set")
            return
        }
        db, err := geoip2.Open(path)
        if err != nil {
            r.err = fmt.Errorf("failed to open GeoIP DB: %w", err)
            return
        }
        r.db = db
    })
    return r.err
}

func (r *resolver) LookupIP(ip string) (string, string, string, error) {
    if err := r.open(); err != nil {
        return "", "", "GeoLite2-City", err
    }
    if r.db == nil {
        return "", "", "GeoLite2-City", errors.New("geoip db not initialized")
    }
    parsed := net.ParseIP(ip)
    if parsed == nil {
        return "", "", "GeoLite2-City", fmt.Errorf("invalid ip: %q", ip)
    }
    rec, err := r.db.City(parsed)
    if err != nil {
        return "", "", "GeoLite2-City", fmt.Errorf("geoip lookup error: %w", err)
    }
    country := rec.Country.IsoCode
    bucket := bucketFrom(rec, country)
    return country, bucket, "GeoLite2-City", nil
}

// bucketFromFields is a small helper used by bucketFrom and unit tests.
func bucketFromFields(continentCode, country string) string {
    // Priority: explicit US -> US
    if country == "US" {
        return "US"
    }
    // Use continent code for broad buckets
    switch continentCode {
    case "EU":
        return "EU"
    case "AS":
        return "ASIA"
    default:
        // For any other continent, we don't have a dedicated bucket yet.
        return "OTHER"
    }
}

func bucketFrom(rec *geoip2.City, country string) string {
    return bucketFromFields(rec.Continent.Code, country)
}
