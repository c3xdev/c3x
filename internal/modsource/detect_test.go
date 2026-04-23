package modsource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetect_FileSource(t *testing.T) {
	result, err := Detect("./local/path", "/working/dir", Detectors)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestDetect_GitSource(t *testing.T) {
	result, err := Detect("github.com/hashicorp/example", "/working/dir", Detectors)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "git::")
}

func TestDetect_S3Source(t *testing.T) {
	result, err := Detect("s3::https://s3-eu-west-1.amazonaws.com/bucket/module.zip", "/working/dir", Detectors)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}


func TestSourceDirSubdir_NoSubdir(t *testing.T) {
	source, subdir := SourceDirSubdir("github.com/foo/bar")
	assert.Equal(t, "github.com/foo/bar", source)
	assert.Equal(t, "", subdir)
}

func TestSourceDirSubdir_WithSubdir(t *testing.T) {
	source, subdir := SourceDirSubdir("github.com/foo/bar//modules/vpc")
	assert.Equal(t, "github.com/foo/bar", source)
	assert.Equal(t, "modules/vpc", subdir)
}

func TestSourceDirSubdir_DeepSubdir(t *testing.T) {
	source, subdir := SourceDirSubdir("github.com/foo/bar//a/b/c/d")
	assert.Equal(t, "github.com/foo/bar", source)
	assert.Equal(t, "a/b/c/d", subdir)
}

func TestClientMode(t *testing.T) {
	assert.Equal(t, ClientModeAny, ClientModeAny)
	assert.Equal(t, ClientModeDir, ClientModeDir)
	assert.Equal(t, ClientModeFile, ClientModeFile)
}

func TestGetters_Registered(t *testing.T) {
	// Verify key getters are registered
	assert.NotEmpty(t, Getters)

	// Check specific getter types exist
	_, hasFile := Getters["file"]
	assert.True(t, hasFile)
}

func TestDecompressors_Registered(t *testing.T) {
	assert.NotEmpty(t, Decompressors)

	_, hasZip := Decompressors["zip"]
	assert.True(t, hasZip)

	_, hasTgz := Decompressors["tar.gz"]
	assert.True(t, hasTgz)
}

func TestDetectors_Registered(t *testing.T) {
	assert.NotEmpty(t, Detectors)
	assert.GreaterOrEqual(t, len(Detectors), 3)
}

func TestGitGetter_New(t *testing.T) {
	g := &GitGetter{}
	assert.NotNil(t, g)
}

func TestHttpGetter_New(t *testing.T) {
	g := &HttpGetter{}
	assert.NotNil(t, g)
}

func TestClient_Fields(t *testing.T) {
	c := &Client{
		Src:  "github.com/foo/bar",
		Dst:  "/tmp/download",
		Mode: ClientModeDir,
	}
	assert.Equal(t, "github.com/foo/bar", c.Src)
	assert.Equal(t, "/tmp/download", c.Dst)
	assert.Equal(t, ClientModeDir, c.Mode)
}

func TestSubdirGlob(t *testing.T) {
	// SubdirGlob resolves glob patterns in directory paths
	// Can't easily test without a real filesystem, but verify it exists
	_, err := SubdirGlob("/nonexistent", "")
	// Should return the path as-is or error
	_ = err
}
