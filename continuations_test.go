package flow

import (
	"fmt"
	"net/textproto"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContinuationReturnsErrorOnPanic(t *testing.T) {
	RegisterAction(thatPanics)
	// assert equality
	result, err := invoke(thatPanics, 12)
	assert.Equal(t, fmt.Errorf("this panicked"), err)
	assert.Empty(t, result)
}

func TestContinuationReturnsErrorOnBadArg(t *testing.T) {
	RegisterAction(toUpperString)
	// assert equality
	result, err := invoke(toUpperString, 12)
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestContinuationPanicsWithoutFunctionArg(t *testing.T) {
	assert.Panics(t, func() {
		RegisterAction("foo")
	})
}

func TestContinuationSuccess(t *testing.T) {
	RegisterAction(toUpperString)
	// assert equality
	result, err := invoke(toUpperString, "foo")
	assert.Equal(t, "FOO", result)
	assert.Nil(t, err)
}

func TestContinuationWithNilError(t *testing.T) {
	RegisterAction(toUpperStringWithNilError)
	// assert equality
	result, err := invoke(toUpperStringWithNilError, "foo")
	assert.Equal(t, "FOO", result)
	assert.Nil(t, err)
}

func TestContinuationWithError(t *testing.T) {
	RegisterAction(toUpperStringWithError)
	// assert equality
	result, err := invoke(toUpperStringWithError, "foo")
	assert.Equal(t, fmt.Errorf("My error"), err)
	assert.Empty(t, result)
}

func TestInvoke(t *testing.T) {
	r, _ := invoke(strings.ToUpper, "foo")
	assert.Equal(t, "FOO", r)
}

func TestDecodeArgString(t *testing.T) {
	r := decodeContinuationArg(strings.ToUpper, 0, encodeGob("foo"), gobArgHeader())
	assert.Equal(t, "foo", r)
	result, err := invoke(strings.ToUpper, r)
	assert.Equal(t, "FOO", result)
	assert.Nil(t, err)
}

type foo struct {
	Name string
}

func testFoo(f *foo) *foo {
	f.Name = strings.ToUpper(f.Name)
	return f
}

func gobArgHeader() *textproto.MIMEHeader {
	hdr := &textproto.MIMEHeader{}
	hdr.Set(DatumTypeHeader, BlobDatumHeader)
	hdr.Set(ContentTypeHeader, GobMediaHeader)
	return hdr
}

func TestDecodeArgWithStruct(t *testing.T) {
	r := decodeContinuationArg(testFoo, 0, encodeGob(&foo{Name: "foo"}), gobArgHeader())
	assert.Equal(t, "foo", r.(*foo).Name)
	result, err := invoke(testFoo, r)
	assert.Equal(t, "FOO", result.(*foo).Name)
	assert.Nil(t, err)
}

func TestEncodeDecodeGob(t *testing.T) {
	e := encodeGob("foo")
	var d string
	d = decodeGob(e, reflect.TypeOf(d)).(string)
	assert.Equal(t, "foo", d)
}

func TestActionIDIsConstant(t *testing.T) {
	k1 := getActionID(TestActionIDIsConstant)
	k2 := getActionID(TestActionIDIsConstant)
	assert.Equal(t, k1, k2)
}

func TestDebugPrints(t *testing.T) {
	Debug(true)
	debug("foo")
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
