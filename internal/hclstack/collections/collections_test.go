package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListContainsElement(t *testing.T) {
	list := []string{"a", "b", "c"}
	assert.True(t, ListContainsElement(list, "b"))
	assert.False(t, ListContainsElement(list, "d"))
}

func TestListContainsElement_Int(t *testing.T) {
	list := []int{1, 2, 3}
	assert.True(t, ListContainsElement(list, 2))
	assert.False(t, ListContainsElement(list, 4))
}

func TestListContainsElement_Empty(t *testing.T) {
	var list []string
	assert.False(t, ListContainsElement(list, "a"))
}

func TestListIntersection(t *testing.T) {
	a := []string{"a", "b", "c"}
	b := []string{"b", "c", "d"}
	result := ListIntersection(a, b)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "b")
	assert.Contains(t, result, "c")
}

func TestListIntersection_NoOverlap(t *testing.T) {
	a := []string{"a", "b"}
	b := []string{"c", "d"}
	result := ListIntersection(a, b)
	assert.Empty(t, result)
}

func TestKeys(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2, "c": 3}
	keys := Keys(m)
	assert.Len(t, keys, 3)
	assert.Contains(t, keys, "a")
	assert.Contains(t, keys, "b")
	assert.Contains(t, keys, "c")
}

func TestKeys_Empty(t *testing.T) {
	m := map[string]int{}
	keys := Keys(m)
	assert.Empty(t, keys)
}

func TestMergeMaps(t *testing.T) {
	m1 := map[string]int{"a": 1, "b": 2}
	m2 := map[string]int{"b": 3, "c": 4}
	result := MergeMaps(m1, m2)

	assert.Equal(t, 1, result["a"])
	assert.Equal(t, 3, result["b"]) // m2 overrides m1
	assert.Equal(t, 4, result["c"])
}

func TestMergeMaps_Empty(t *testing.T) {
	result := MergeMaps[string, int]()
	assert.Empty(t, result)
}
