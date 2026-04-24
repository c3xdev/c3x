package util

import (
	"bytes"
	"math/rand"
	"time"
)

// GetRandomTime returns a random duration between lowerBound and upperBound (inclusive).
// Negative bounds are converted to their absolute values. If bounds are equal, that value is returned.
// If lowerBound > upperBound, the bounds are swapped.
func GetRandomTime(lowerBound, upperBound time.Duration) time.Duration {
	if lowerBound < 0 {
		lowerBound = -lowerBound
	}
	if upperBound < 0 {
		upperBound = -upperBound
	}
	if lowerBound > upperBound {
		lowerBound, upperBound = upperBound, lowerBound
	}
	if lowerBound == upperBound {
		return lowerBound
	}

	rangeMs := upperBound.Milliseconds() - lowerBound.Milliseconds()
	offset := rand.Int63n(rangeMs + 1) //nolint:gosec // non-cryptographic use
	return lowerBound + time.Duration(offset)*time.Millisecond
}

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
const uniqueIDLength = 6 // 62^6 = 56+ billion combinations

// UniqueId returns a random 6-character base62 string for use in naming test resources.
func UniqueId() string {
	var out bytes.Buffer
	for i := 0; i < uniqueIDLength; i++ {
		out.WriteByte(base62Chars[rand.Intn(len(base62Chars))]) //nolint:gosec // non-cryptographic use
	}
	return out.String()
}
