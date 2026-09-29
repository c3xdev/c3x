package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeAZProject writes a configuration with one instance per placeholder
// availability zone, one with a literal count, and a free resource whose
// count also rests on the zones.
func writeAZProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	tf := `
		provider "aws" { region = "us-east-1" }
		data "aws_availability_zones" "available" {}
		resource "aws_instance" "per_az" {
		  count         = length(data.aws_availability_zones.available.names)
		  instance_type = "t3.micro"
		}
		resource "aws_instance" "fixed" {
		  count         = 2
		  instance_type = "t3.micro"
		}
		resource "aws_vpc" "per_az" {
		  count      = length(data.aws_availability_zones.available.names)
		  cidr_block = "10.0.0.0/16"
		}
	`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(tf), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// An instance count computed from a data source placeholder is an
// assumed_count caveat on each priced instance, not on a free resource
// nor on a literal count.
func TestAssumedCountCaveatInJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	out, err := runCLI(t, "estimate", "--path", writeAZProject(t), "--format", "json",
		"--pricing-endpoint", flatPricing(t), "--no-cache")
	if err != nil {
		t.Fatalf("estimate: %v\n%s", err, out)
	}
	start := strings.Index(out, "{")
	if start < 0 {
		t.Fatalf("no JSON in output:\n%s", out)
	}
	var got struct {
		CaveatCount int `json:"caveat_count"`
		Costs       []struct {
			Resource string `json:"resource"`
			Subtotal string `json:"monthly_subtotal"`
			Caveats  []struct {
				Code   string `json:"code"`
				Detail string `json:"detail"`
			} `json:"caveats"`
		} `json:"costs"`
	}
	if err := json.NewDecoder(strings.NewReader(out[start:])).Decode(&got); err != nil {
		t.Fatalf("decode: %v\n%s", err, out)
	}
	assumed := map[string]string{}
	seen := map[string]bool{}
	for _, c := range got.Costs {
		seen[c.Resource] = true
		for _, cv := range c.Caveats {
			if cv.Code == "assumed_count" {
				assumed[c.Resource] = cv.Detail
			}
		}
	}
	want := "instance count assumes data.aws_availability_zones.available.names = [us-east-1a, us-east-1b, us-east-1c]; price a plan JSON for exact counts"
	for _, r := range []string{"aws_instance.per_az[0]", "aws_instance.per_az[1]", "aws_instance.per_az[2]"} {
		if assumed[r] != want {
			t.Errorf("%s: assumed_count detail = %q, want %q", r, assumed[r], want)
		}
	}
	for _, r := range []string{"aws_instance.fixed[0]", "aws_instance.fixed[1]", "aws_vpc.per_az[0]"} {
		if d, ok := assumed[r]; ok {
			t.Errorf("%s has an assumed_count caveat (%q); want none", r, d)
		}
	}
	if !seen["aws_instance.fixed[0]"] {
		t.Errorf("aws_instance.fixed[0] missing from costs: %v", seen)
	}
	if got.CaveatCount < 3 {
		t.Errorf("caveat_count = %d; the three assumed counts must count", got.CaveatCount)
	}
}

// The caveat shows in the text breakdown and the markdown PR comment,
// and --strict fails on it with exit code 3.
func TestAssumedCountCaveatRendersAndFailsStrict(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := writeAZProject(t)
	endpoint := flatPricing(t)
	for _, format := range []string{"text", "markdown"} {
		out, err := runCLI(t, "estimate", "--path", dir, "--format", format,
			"--pricing-endpoint", endpoint, "--no-cache")
		if err != nil {
			t.Fatalf("%s: %v\n%s", format, err, out)
		}
		want := "instance count assumes data.aws_availability_zones.available.names"
		if n := strings.Count(out, want); n != 3 {
			t.Errorf("%s output mentions the assumed count %d times, want 3 (the priced instances only):\n%s", format, n, out)
		}
	}

	out, err := runCLI(t, "estimate", "--path", dir, "--strict",
		"--pricing-endpoint", endpoint, "--no-cache")
	if !errors.Is(err, errStrictCaveats) {
		t.Fatalf("--strict err = %v, want errStrictCaveats\n%s", err, out)
	}
	if !strings.Contains(out, "assumed_count") {
		t.Errorf("--strict summary lacks assumed_count:\n%s", out)
	}
}
