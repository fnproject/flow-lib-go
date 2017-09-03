package completions

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContinuationReturnsErrorOnPanic(t *testing.T) {
	RegisterContinuation(thatPanics)
	// assert equality
	result, err := invoke(thatPanics, 12)
	assert.Equal(t, fmt.Errorf("this panicked"), err)
	assert.Empty(t, result)
}

func TestContinuationReturnsErrorOnBadArg(t *testing.T) {
	RegisterContinuation(toUpperString)
	// assert equality
	result, err := invoke(toUpperString, 12)
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestContinuationPanicsWithoutFunctionArg(t *testing.T) {
	assert.Panics(t, func() {
		RegisterContinuation("foo")
	})
}

func TestContinuationSuccess(t *testing.T) {
	RegisterContinuation(toUpperString)
	// assert equality
	result, err := invoke(toUpperString, "foo")
	assert.Equal(t, "FOO", result)
	assert.Nil(t, err)
}

func TestContinuationWithNilError(t *testing.T) {
	RegisterContinuation(toUpperStringWithNilError)
	// assert equality
	result, err := invoke(toUpperStringWithNilError, "foo")
	assert.Equal(t, "FOO", result)
	assert.Nil(t, err)
}

func TestContinuationWithError(t *testing.T) {
	RegisterContinuation(toUpperStringWithError)
	// assert equality
	result, err := invoke(toUpperStringWithError, "foo")
	assert.Equal(t, fmt.Errorf("My error"), err)
	assert.Empty(t, result)
}

func toUpperString(arg0 string) string {
	return strings.ToUpper(arg0)
}

func toUpperStringWithError(arg0 string) (string, error) {
	return "", fmt.Errorf("My error")
}

func toUpperStringWithNilError(arg0 string) (string, error) {
	return strings.ToUpper(arg0), nil
}

func thatPanics(v int) {
	panic("this panicked")
}