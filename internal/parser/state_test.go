package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/c3xdev/c3x/internal/parser"
)

func TestParseStateReadsAStateFile(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	state := `{"values":{"root_module":{"resources":[{
		"address":"aws_s3_bucket.data","mode":"managed","type":"aws_s3_bucket","name":"data",
		"values":{"bucket":"acme-data","region":"ap-northeast-1"}}]}}}`
	if err := os.WriteFile(path, []byte(state), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := parser.ParseState(path, parser.Options{})
	if err != nil {
		t.Fatalf("ParseState: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 resource, got %d", len(got))
	}
	if got[0].Attributes["bucket"] != "acme-data" {
		t.Errorf("bucket = %v", got[0].Attributes["bucket"])
	}
	if got[0].Region == nil || *got[0].Region != "ap-northeast-1" {
		t.Errorf("region = %v, want ap-northeast-1", got[0].Region)
	}
}

func TestParseStateRejectsMissingFile(t *testing.T) {
	t.Parallel()

	if _, err := parser.ParseState(filepath.Join(t.TempDir(), "nope.json"), parser.Options{}); err == nil {
		t.Fatal("want an error for a missing file")
	}
}
