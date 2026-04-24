// Copyright (c) C3X Dev

//go:build !windows
// +build !windows

package url

import (
	"net/url"
)

func parse(rawURL string) (*url.URL, error) {
	return url.Parse(rawURL)
}
