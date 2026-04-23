// Package collections provides generic collection utility functions.
package collections

import "fmt"

// ListContainsElement returns true if the list contains the given element.
func ListContainsElement[T comparable](list []T, element T) bool {
	for _, item := range list {
		if item == element {
			return true
		}
	}
	return false
}

// ListIntersection returns elements that are in both lists.
func ListIntersection[T comparable](a, b []T) []T {
	set := make(map[T]bool)
	for _, item := range b {
		set[item] = true
	}
	var result []T
	for _, item := range a {
		if set[item] {
			result = append(result, item)
		}
	}
	return result
}

// Keys returns the keys of a map.
func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// MergeMaps merges multiple maps into one. Later maps override earlier ones.
func MergeMaps[K comparable, V any](maps ...map[K]V) map[K]V {
	result := make(map[K]V)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

// MapJoin joins a map into a string with the given argument and value separators.
// Example: MapJoin({"a": "1", "b": "2"}, ",", "=") → "a=1,b=2"
func MapJoin[K comparable, V any](m map[K]V, argSep string, valSep string) string {
	result := ""
	for k, v := range m {
		if result != "" {
			result += argSep
		}
		result += fmt.Sprintf("%v%s%v", k, valSep, v)
	}
	return result
}
