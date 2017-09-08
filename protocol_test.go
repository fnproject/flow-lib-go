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
	buf := encodeGob(25)
	result := decodeTypedGob(buf, reflect.TypeOf(25))
	assert.Equal(t, 25, result)
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
	assert.Panics(t, func() {
		continuationArgTypes(withThreeArgs)
	})
}

func TestDecodeContinuationArgsWithOne(t *testing.T) {
	args := decodeContinuationArgs(withOneArg, encodeGob("foo"))
	assert.Equal(t, 1, len(args))
	assert.Equal(t, "foo", args[0])
}

func TestDecodeContinuationArgsWithTwo(t *testing.T) {
	args := decodeContinuationArgs(withTwoArgs, encodeGob("foo"), encodeGob(25))
	assert.Equal(t, 2, len(args))
	assert.Equal(t, "foo", args[0])
	assert.IsType(t, 25, args[1])
}

func TestDecodeContinuationArgsThatFails(t *testing.T) {
	assert.Panics(t, func() {
		decodeContinuationArgs(withOneArg, encodeGob("foo"), encodeGob(25))
	})
}

func withOneArg(one string) {
}

func withTwoArgs(one string, two int) {
}

func withThreeArgs(one string, two int, e error) {
}
