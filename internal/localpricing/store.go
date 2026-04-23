// Package localpricing provides a local SQLite-based pricing database for offline cost estimation.
package localpricing

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS products (
	vendor TEXT NOT NULL,
	region TEXT NOT NULL,
	service TEXT NOT NULL,
	product_family TEXT NOT NULL DEFAULT '',
	sku TEXT NOT NULL,
	attributes TEXT NOT NULL DEFAULT '{}',
	prices TEXT NOT NULL DEFAULT '[]',
	PRIMARY KEY (vendor, sku)
);

CREATE INDEX IF NOT EXISTS idx_products_vendor_service ON products(vendor, service);
CREATE INDEX IF NOT EXISTS idx_products_vendor_region ON products(vendor, region);

CREATE TABLE IF NOT EXISTS metadata (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// Store represents a local SQLite pricing database.
type Store struct {
	db   *sql.DB
	path string
}

// DefaultPath returns the default path for the local pricing database.
func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".c3x", "pricing.db")
}

// Open opens or creates a local pricing database.
func Open(path string) (*Store, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating pricing database directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening pricing database: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("initializing pricing schema: %w", err)
	}

	return &Store{db: db, path: path}, nil
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}

// Exists returns true if the pricing database file exists and has data.
func Exists(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return false
	}
	s, err := Open(path)
	if err != nil {
		return false
	}
	defer s.Close()
	var dummy int
	err = s.db.QueryRow("SELECT 1 FROM products LIMIT 1").Scan(&dummy)
	return err == nil
}

// UpsertProduct inserts or updates a product in the local database.
func (s *Store) UpsertProduct(vendor, region, service, productFamily, sku, attributes, prices string) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO products (vendor, region, service, product_family, sku, attributes, prices)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		vendor, region, service, productFamily, sku, attributes, prices,
	)
	return err
}

// SetMetadata sets a metadata key-value pair.
func (s *Store) SetMetadata(key, value string) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`,
		key, value,
	)
	return err
}

// GetMetadata gets a metadata value by key.
func (s *Store) GetMetadata(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM metadata WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// ProductCount returns the total number of products in the database.
func (s *Store) ProductCount() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&count)
	return count, err
}

// QueryProducts queries products matching the given filters.
func (s *Store) QueryProducts(vendor, service, region string, attributeFilters map[string]string) (*sql.Rows, error) {
	query := `SELECT sku, attributes, prices FROM products WHERE vendor = ? AND service = ?`
	args := []interface{}{vendor, service}

	if region != "" {
		query += ` AND region = ?`
		args = append(args, region)
	}

	for key, value := range attributeFilters {
		query += ` AND json_extract(attributes, ?) = ?`
		args = append(args, "$."+key, value)
	}

	return s.db.Query(query, args...)
}
