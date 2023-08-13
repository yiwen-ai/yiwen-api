package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSliceHas(t *testing.T) {
	assert.True(t, SliceHas([]string{"a", "b", "c"}, "a"))
	assert.False(t, SliceHas([]string{"a", "b", "c"}, "d"))

	assert.True(t, SliceHas([]int{1, 2, 3}, 1))
	assert.False(t, SliceHas([]int{1, 2, 3}, 0))
}

func TestRemoveDuplicates(t *testing.T) {
	assert.Equal(t, RemoveDuplicates([]string{"a", "b", "a"}), []string{"a", "b"})

	id := NewID()
	id2 := mustParseID(id.String())
	assert.Equal(t, RemoveDuplicates([]ID{id, id2}), []ID{id})
}
