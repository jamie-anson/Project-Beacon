//go:build ignore
// +build ignore

package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	beaconcrypto "github.com/jamie-anson/project-beacon-runner/pkg/crypto"
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

	// Zero signature fields before signing
	spec.Signature = ""
	spec.PublicKey = ""


	// Make signable view
	signable, err := beaconcrypto.CreateSignableJobSpec(&spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create signable view: %v\n", err)
		os.Exit(1)
	}

	// Generate ephemeral keypair and sign
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate key: %v\n", err)
		os.Exit(1)
	}
	sig, err := beaconcrypto.SignJSON(signable, priv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to sign: %v\n", err)
		os.Exit(1)
	}

	// Inject signature and public key
	spec.Signature = sig
	spec.PublicKey = base64.StdEncoding.EncodeToString(pub)

	outBytes, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal signed jobspec: %v\n", err)
		os.Exit(1)
	}
	if err := ioutil.WriteFile(*out, outBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("wrote signed jobspec to %s\n", *out)
}
