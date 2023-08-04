package utils_test

import (
	"testing"

	"github.com/cnrancher/cce-operator/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func Test_PrintObject(t *testing.T) {
	assert.Equal(t, utils.PrintObject(nil), "null")
	assert.Equal(t, utils.PrintObject(struct{}{}), "{}")
	assert.Equal(t,
		utils.PrintObject(map[string]string{"key": "value"}), `{
  "key": "value"
}`)
}

func Test_Parse(t *testing.T) {
	ns, n := utils.Parse("")
	assert.Equal(t, ns, "")
	assert.Equal(t, n, "")
	ns, n = utils.Parse("name")
	assert.Equal(t, ns, "")
	assert.Equal(t, n, "name")
	ns, n = utils.Parse("cattle-global-data:name")
	assert.Equal(t, ns, "cattle-global-data")
	assert.Equal(t, n, "name")
}

func Test_RandomHex(t *testing.T) {
	a := utils.RandomHex(5)
	assert.Equal(t, len(a), 5)
}

func Test_Pointer(t *testing.T) {
	assert.Equal(t, *utils.Pointer("a"), "a")
	assert.Equal(t, *utils.Pointer(int(1)), int(1))
	assert.Equal(t, *utils.Pointer(int32(1)), int32(1))
	assert.Equal(t, *utils.Pointer(float32(1)), float32(1))
	assert.Equal(t, *utils.Pointer(1), 1)
}

func Test_Value(t *testing.T) {
	var int32Ptr *int32
	assert.Equal(t, utils.Value(int32Ptr), int32(0))
	int32Ptr = new(int32)
	*int32Ptr = 123
	assert.Equal(t, utils.Value(int32Ptr), int32(123))

	var stringPtr *string
	assert.Equal(t, utils.Value(stringPtr), "")
	stringPtr = new(string)
	*stringPtr = "hello"
	assert.Equal(t, utils.Value(stringPtr), "hello")
}
