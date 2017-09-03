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
	assert.Panics(t, func() {
		continuationArgTypes(withThreeArgs)
	})
}

func TestDecodeContinuationArgsWithOne(t *testing.T) {
	stringArg := "foo"
	//bufs := []*bytes.Buffer{encodeGob(&stringArg)}
	args := decodeContinuationArgs(withOneArg, encodeGob(&stringArg))
	assert.Equal(t, 1, len(args))
	assert.IsType(t, &stringArg, args[0])
}

func TestDecodeContinuationArgsWithTwo(t *testing.T) {
	stringArg := "foo"
	intArg := 25
	args := decodeContinuationArgs(withTwoArgs, encodeGob(&stringArg), encodeGob(&intArg))
	assert.Equal(t, 2, len(args))
	assert.IsType(t, &stringArg, args[0])
	assert.IsType(t, &intArg, args[1])
}

func TestDecodeContinuationArgsThatFails(t *testing.T) {
	stringArg := "foo"
	intArg := 25
	assert.Panics(t, func() {
		decodeContinuationArgs(withOneArg, encodeGob(&stringArg), encodeGob(&intArg))
	})
}

func withOneArg(one string) {
}

func withTwoArgs(one string, two int) {
}

func withThreeArgs(one string, two int, e error) {
}
