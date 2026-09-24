package terraform_test

import (
	"reflect"
	"testing"

	"github.com/c3xdev/c3x/internal/parser/terraform"
)

// parseOne parses a one-file config and returns its resources by name.
func parseOne(t *testing.T, src string) map[string]map[string]any {
	t.Helper()
	dir := t.TempDir()
	write(t, dir, "main.tf", src)
	got, err := terraform.ParseDirectory(dir, terraform.Options{Offline: true})
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]map[string]any{}
	for _, r := range got {
		out[r.Ref.Name] = r.Attributes
	}
	return out
}

// A dynamic block produces exactly the attributes the equivalent literal
// blocks do. It used to surface as an attribute named "dynamic", so the
// EBS volumes it declared were never priced.
func TestDynamicBlockMatchesLiteralBlocks(t *testing.T) {
	t.Parallel()
	rs := parseOne(t, `
		variable "disks" {
		  default = [
		    { size = 100, type = "gp3" },
		    { size = 200, type = "io2" },
		  ]
		}
		resource "aws_instance" "literal" {
		  instance_type = "m5.large"
		  ebs_block_device {
		    volume_size = 100
		    volume_type = "gp3"
		  }
		  ebs_block_device {
		    volume_size = 200
		    volume_type = "io2"
		  }
		}
		resource "aws_instance" "dynamic" {
		  instance_type = "m5.large"
		  dynamic "ebs_block_device" {
		    for_each = var.disks
		    content {
		      volume_size = ebs_block_device.value.size
		      volume_type = ebs_block_device.value.type
		    }
		  }
		}
	`)
	if _, bad := rs["dynamic"]["dynamic"]; bad {
		t.Fatal(`dynamic block surfaced as an attribute named "dynamic"`)
	}
	if !reflect.DeepEqual(rs["dynamic"], rs["literal"]) {
		t.Errorf("dynamic = %#v\nliteral = %#v", rs["dynamic"], rs["literal"])
	}
}

// One element gives the single-block shape (a map), zero gives nothing,
// a custom iterator name works, and key is the list index.
func TestDynamicBlockShapesAndIterator(t *testing.T) {
	t.Parallel()
	rs := parseOne(t, `
		resource "aws_instance" "one" {
		  dynamic "root_block_device" {
		    for_each = [50]
		    iterator = disk
		    content {
		      volume_size = disk.value
		      index       = disk.key
		    }
		  }
		}
		resource "aws_instance" "none" {
		  dynamic "ebs_block_device" {
		    for_each = []
		    content { volume_size = 10 }
		  }
		}
	`)
	want := map[string]any{"volume_size": float64(50), "index": float64(0)}
	if got := rs["one"]["root_block_device"]; !reflect.DeepEqual(got, want) {
		t.Errorf("root_block_device = %#v, want %#v", got, want)
	}
	if got, ok := rs["none"]["ebs_block_device"]; ok {
		t.Errorf("empty for_each produced ebs_block_device = %#v; want none", got)
	}
}

// Nested dynamic blocks expand with the outer iterator in scope, and mix
// with literal blocks of the same type in source order.
func TestNestedDynamicBlocks(t *testing.T) {
	t.Parallel()
	rs := parseOne(t, `
		locals {
		  groups = { a = ["x", "y"], b = ["z"] }
		}
		resource "google_compute_instance" "vm" {
		  disk { name = "boot" }
		  dynamic "disk" {
		    for_each = local.groups
		    content {
		      name = disk.key
		      dynamic "label" {
		        for_each = disk.value
		        content { v = "${disk.key}-${label.value}" }
		      }
		    }
		  }
		}
	`)
	want := []any{
		map[string]any{"name": "boot"},
		map[string]any{"name": "a", "label": []any{
			map[string]any{"v": "a-x"},
			map[string]any{"v": "a-y"},
		}},
		map[string]any{"name": "b", "label": map[string]any{"v": "b-z"}},
	}
	if got := rs["vm"]["disk"]; !reflect.DeepEqual(got, want) {
		t.Errorf("disk = %#v\nwant %#v", got, want)
	}
}
