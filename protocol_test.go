package completions

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeAndDecodeGob(t *testing.T) {
	buf := encodeGob(25)
	var result int
	decodeGob(buf, &result)
	assert.Equal(t, 25, result)
}

func TestEncodeAndDecodeGobInterface(t *testing.T) {
	v := 25
	buf := encodeGob(&v)
	result := decodeTypedGob(buf, reflect.TypeOf(v))
	assert.Equal(t, &v, result)
}

func TestContinuationTypesOneArg(t *testing.T) {
	argTypes := continuationArgTypes(TestContinuationTypesOneArg)
	assert.Equal(t, reflect.TypeOf(t), argTypes[0])
	assert.Equal(t, 1, len(argTypes))
}

func TestContinuationTypesTwoArgs(t *testing.T) {
	argTypes := continuationArgTypes(withTwoArgs)
	assert.Equal(t, reflect.TypeOf("string"), argTypes[0])
	assert.Equal(t, reflect.TypeOf(0), argTypes[1])
	assert.Equal(t, 2, len(argTypes))
}

func TestContinuationTypesExceedsArgs(t *testing.T) {
	assert.Panics(t, func() { continuationArgTypes(withThreeArgs) })
}

func withTwoArgs(one string, two int) {
}

func withThreeArgs(one string, two int, e error) {
}
