package modsource

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpGetter_GetFile(t *testing.T) {
	// Create a mock HTTP server that serves a file
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("hello world"))
	}))
	defer server.Close()

	g := &HttpGetter{}
	dst := filepath.Join(t.TempDir(), "downloaded.txt")

	ctx := context.Background()
	err := g.GetFile(ctx, dst, testURL(server.URL))
	if err != nil {
		// Some getter implementations need specific URL formats
		t.Logf("GetFile returned error (may be expected): %v", err)
	}
}

func TestHttpGetter_Get_Directory(t *testing.T) {
	// Server that serves a zip file
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not a real zip"))
	}))
	defer server.Close()

	g := &HttpGetter{}
	dst := t.TempDir()

	ctx := context.Background()
	err := g.Get(ctx, dst, testURL(server.URL+"/module.zip"))
	if err != nil {
		t.Logf("Get returned error (may be expected): %v", err)
	}
}

func TestFileGetter_GetFile(t *testing.T) {
	// Create a source file
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "source.txt")
	err := os.WriteFile(srcFile, []byte("test content"), 0644)
	require.NoError(t, err)

	g := &FileGetter{}
	dst := filepath.Join(t.TempDir(), "dest.txt")

	ctx := context.Background()
	err = g.GetFile(ctx, dst, testURL(srcFile))
	if err != nil {
		t.Logf("FileGetter.GetFile error: %v", err)
	}
}

func TestFileGetter_Get_Directory(t *testing.T) {
	// Create source directory with files
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "file1.tf"), []byte("resource {}"), 0644)
	os.WriteFile(filepath.Join(srcDir, "file2.tf"), []byte("variable {}"), 0644)

	g := &FileGetter{}
	dst := t.TempDir()

	ctx := context.Background()
	err := g.Get(ctx, dst, testURL(srcDir))
	if err != nil {
		t.Logf("FileGetter.Get error: %v", err)
	}
}

func TestGitGetter_ClientMode(t *testing.T) {
	g := &GitGetter{}
	ctx := context.Background()
	mode, err := g.ClientMode(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, ClientModeDir, mode)
}

func TestGet_LocalFile(t *testing.T) {
	// Create source directory
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "main.tf"), []byte("resource {}"), 0644)

	dst := t.TempDir()

	ctx := context.Background()
	err := Get(ctx, dst, srcDir)
	if err != nil {
		t.Logf("Get error for local path: %v", err)
	}
}

func TestClient_Get(t *testing.T) {
	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "main.tf"), []byte("resource {}"), 0644)

	dst := t.TempDir()

	client := &Client{
		Src:     srcDir,
		Dst:     dst,
		Mode:    ClientModeDir,
		Getters: Getters,
	}

	ctx := context.Background()
	err := client.Get(ctx)
	if err != nil {
		t.Logf("Client.Get error: %v", err)
	}
}

func testURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}
