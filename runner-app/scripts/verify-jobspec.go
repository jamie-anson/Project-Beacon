//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/jamie-anson/project-beacon-runner/pkg/models"
)

func main() {
	in := flag.String("in", "", "input signed jobspec JSON")
	flag.Parse()
	if *in == "" {
		fmt.Fprintln(os.Stderr, "-in path is required")
		os.Exit(2)
	}
	b, err := ioutil.ReadFile(*in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		os.Exit(1)
	}
	var spec models.JobSpec
	if err := json.Unmarshal(b, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "unmarshal error: %v\n", err)
		os.Exit(1)
	}
	v := models.NewJobSpecValidator()
	if err := v.ValidateAndVerify(&spec); err != nil {
		fmt.Printf("VERIFY: FAIL: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("VERIFY: OK")
}
