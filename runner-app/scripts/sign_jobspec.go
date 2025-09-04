//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"crypto/ed25519"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func main() {
	in := "jobspec.json"
	if len(os.Args) > 1 {
		in = os.Args[1]
	}
	out := "/tmp/signed_jobspec.json"
	if len(os.Args) > 2 {
		out = os.Args[2]
	}

	data, err := ioutil.ReadFile(in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", in, err)
		os.Exit(1)
	}

	var js models.JobSpec
	if err := json.Unmarshal(data, &js); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal: %v\n", err)
		os.Exit(1)
	}

	// Ensure required fields and apply defaults BEFORE signing
	if js.CreatedAt.IsZero() {
		js.CreatedAt = time.Now()
	}
	if js.Version == "" {
		js.Version = "v0.1.0"
	}

	validator := models.NewJobSpecValidator()
	validator.SanitizeJobSpec(&js)
	if err := js.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "pre-sign validate: %v\n", err)
		os.Exit(1)
	}

	// Generate ephemeral ed25519 keypair for demo signing
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate key: %v\n", err)
		os.Exit(1)
	}

	// Sign the JobSpec (sets PublicKey and Signature)
	if err := js.Sign(priv); err != nil {
		fmt.Fprintf(os.Stderr, "sign: %v\n", err)
		os.Exit(1)
	}

	// Validate & verify with our validator to be safe
	if err := validator.ValidateAndVerify(&js); err != nil {
		fmt.Fprintf(os.Stderr, "validate+verify: %v\n", err)
		os.Exit(1)
	}

	outBytes, err := json.MarshalIndent(js, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal signed: %v\n", err)
		os.Exit(1)
	}
	if err := ioutil.WriteFile(out, outBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", out, err)
		os.Exit(1)
	}

	fmt.Printf("Signed jobspec written to %s\n", out)
	fmt.Printf("PublicKey: %s\n", js.PublicKey)
	fmt.Printf("Signature: %s\n", js.Signature)
}
