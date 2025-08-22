//go:build ignore
// +build ignore

package main

import (
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func main() {
	in := flag.String("in", "", "input jobspec JSON (.signed or unsigned)")
	out := flag.String("out", "", "output signed jobspec JSON")
	newID := flag.String("id", "", "override job id (optional)")
	newImage := flag.String("image", "", "override container image (optional)")
	newTag := flag.String("tag", "", "override container tag (optional)")
	flag.Parse()

	if *in == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: -in input.json -out output.json [-id new-id] [-image repo/name] [-tag tag]")
		os.Exit(2)
	}

	b, err := ioutil.ReadFile(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read input: %v\n", err)
		os.Exit(1)
	}

	var spec models.JobSpec
	if err := json.Unmarshal(b, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse jobspec: %v\n", err)
		os.Exit(1)
	}

	// Apply overrides
	if *newID != "" {
		spec.ID = *newID
	}
	if *newImage != "" {
		spec.Benchmark.Container.Image = *newImage
	}
	if *newTag != "" {
		spec.Benchmark.Container.Tag = *newTag
	}

	// Ensure fields match server preprocessing before signing
	if spec.CreatedAt.IsZero() {
		spec.CreatedAt = time.Now()
	}
	if spec.Version == "" {
		spec.Version = "v0.1.0"
	}
	// Mirror defaults applied in models.JobSpec.Validate() which server calls BEFORE VerifySignature
	if spec.Constraints.MinSuccessRate == 0 {
		spec.Constraints.MinSuccessRate = 0.67
	}
	if spec.Constraints.ProviderTimeout == 0 {
		spec.Constraints.ProviderTimeout = 2 * time.Minute
	}

	validator := models.NewJobSpecValidator()
	validator.SanitizeJobSpec(&spec)

	// Validate prior to signing (server does this as well)
	if err := spec.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "pre-sign validate failed: %v\n", err)
		os.Exit(1)
	}

	// Zero signature fields before signing
	spec.Signature = ""
	spec.PublicKey = ""

	// Generate ephemeral keypair and sign using the same method as server
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate key: %v\n", err)
		os.Exit(1)
	}
	if err := spec.Sign(priv); err != nil {
		fmt.Fprintf(os.Stderr, "failed to sign jobspec: %v\n", err)
		os.Exit(1)
	}

	// Safety check: self-verify before writing
	if err := spec.VerifySignature(); err != nil {
		fmt.Fprintf(os.Stderr, "self-verify failed: %v\n", err)
		os.Exit(1)
	}

	outBytes, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal signed jobspec: %v\n", err)
		os.Exit(1)
	}
	if err := ioutil.WriteFile(*out, outBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}

	// Confirm output and show key material (public key already embedded by Sign())
	fmt.Printf("wrote signed jobspec to %s\n", *out)
	fmt.Printf("PublicKey: %s\n", spec.PublicKey)
	fmt.Printf("Signature: %s\n", spec.Signature)
}
