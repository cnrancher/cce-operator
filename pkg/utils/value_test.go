package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_GetValue(t *testing.T) {
	var int32Ptr *int32
	assert.Equal(t, GetValue(int32Ptr), int32(0))
	int32Ptr = new(int32)
	*int32Ptr = 123
	assert.Equal(t, GetValue(int32Ptr), int32(123))

	var stringPtr *string
	assert.Equal(t, GetValue(stringPtr), "")
	stringPtr = new(string)
	*stringPtr = "hello"
	assert.Equal(t, GetValue(stringPtr), "hello")
}
