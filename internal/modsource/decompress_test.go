package modsource

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestZipDecompressor(t *testing.T) {
	d := new(ZipDecompressor)
	assert.NotNil(t, d)
}

func TestTarGzipDecompressor(t *testing.T) {
	d := new(TarGzipDecompressor)
	assert.NotNil(t, d)
}

func TestTarBzip2Decompressor(t *testing.T) {
	d := new(TarBzip2Decompressor)
	assert.NotNil(t, d)
}

func TestTarXzDecompressor(t *testing.T) {
	d := new(TarXzDecompressor)
	assert.NotNil(t, d)
}

func TestGzipDecompressor(t *testing.T) {
	d := new(GzipDecompressor)
	assert.NotNil(t, d)
}

func TestBzip2Decompressor(t *testing.T) {
	d := new(Bzip2Decompressor)
	assert.NotNil(t, d)
}

func TestXzDecompressor(t *testing.T) {
	d := new(XzDecompressor)
	assert.NotNil(t, d)
}

func TestTarDecompressor(t *testing.T) {
	d := new(TarDecompressor)
	assert.NotNil(t, d)
}


func TestFolderStorage(t *testing.T) {
	// FileGetter downloads to a folder
	g := new(FileGetter)
	assert.NotNil(t, g)
}

func TestS3Getter(t *testing.T) {
	g := new(S3Getter)
	assert.NotNil(t, g)
}

func TestGCSGetter(t *testing.T) {
	g := new(GCSGetter)
	assert.NotNil(t, g)
}



func TestHgGetter(t *testing.T) {
	g := new(HgGetter)
	assert.NotNil(t, g)
}
