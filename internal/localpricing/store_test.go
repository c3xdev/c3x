package localpricing

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_CreatesDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	require.NoError(t, err)
	defer store.Close()

	count, err := store.ProductCount()
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestUpsertProduct_And_ProductCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	require.NoError(t, err)
	defer store.Close()

	err = store.UpsertProduct("aws", "us-east-1", "AmazonEC2", "Compute Instance", "SKU123",
		`{"instanceType":"m5.xlarge"}`, `[{"USD":"0.192","unit":"Hrs"}]`)
	require.NoError(t, err)

	count, err := store.ProductCount()
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestUpsertProduct_Upsert(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	require.NoError(t, err)
	defer store.Close()

	// Insert
	err = store.UpsertProduct("aws", "us-east-1", "AmazonEC2", "Compute", "SKU1",
		`{"instanceType":"m5.xlarge"}`, `[{"USD":"0.192"}]`)
	require.NoError(t, err)

	// Upsert with new price
	err = store.UpsertProduct("aws", "us-east-1", "AmazonEC2", "Compute", "SKU1",
		`{"instanceType":"m5.xlarge"}`, `[{"USD":"0.200"}]`)
	require.NoError(t, err)

	count, err := store.ProductCount()
	require.NoError(t, err)
	assert.Equal(t, 1, count) // Still 1, not 2
}

func TestMetadata(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	require.NoError(t, err)
	defer store.Close()

	err = store.SetMetadata("last_sync", "2026-04-10T12:00:00Z")
	require.NoError(t, err)

	val, err := store.GetMetadata("last_sync")
	require.NoError(t, err)
	assert.Equal(t, "2026-04-10T12:00:00Z", val)
}

func TestGetMetadata_Missing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	require.NoError(t, err)
	defer store.Close()

	val, err := store.GetMetadata("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestExists_False(t *testing.T) {
	assert.False(t, Exists("/tmp/nonexistent_c3x_pricing.db"))
}

func TestExists_True(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	require.NoError(t, err)
	store.UpsertProduct("aws", "us-east-1", "AmazonEC2", "Compute", "SKU1", `{}`, `[]`)
	store.Close()

	assert.True(t, Exists(path))
}

func TestLookupPrice_Found(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	require.NoError(t, err)
	defer store.Close()

	store.UpsertProduct("aws", "us-east-1", "AmazonEC2", "Compute", "SKU1",
		`{"instanceType":"m5.xlarge","operatingSystem":"Linux"}`,
		`[{"USD":"0.192","unit":"Hrs"}]`)

	result, err := store.LookupPrice("aws", "AmazonEC2", "us-east-1",
		map[string]string{"instanceType": "m5.xlarge"}, "")
	require.NoError(t, err)
	assert.Contains(t, result.Raw, "0.192")
}

func TestLookupPrice_NotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(path)
	require.NoError(t, err)
	defer store.Close()

	result, err := store.LookupPrice("aws", "AmazonEC2", "us-east-1",
		map[string]string{"instanceType": "nonexistent.xlarge"}, "")
	require.NoError(t, err)
	assert.Empty(t, result.Raw)
}
