package completions

import (
	"errors"
	"net/textproto"
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

func TestDecodeContinuationArgsWithOneGob(t *testing.T) {
	arg := decodeContinuationArg(withOneArg, 0, encodeGob("foo"), gobHeaders())
	assert.Equal(t, "foo", arg)
}

func TestDecodeContinuationArgsWithTwoGobs(t *testing.T) {
	arg := decodeContinuationArg(withTwoArgs, 0, encodeGob("foo"), gobHeaders())
	assert.Equal(t, "foo", arg)
	arg = decodeContinuationArg(withTwoArgs, 1, encodeGob(25), gobHeaders())
	assert.IsType(t, 25, arg)
}

func TestDecodeContinuationArgsThatFails(t *testing.T) {
	assert.Panics(t, func() {
		decodeContinuationArg(withOneArg, 2, encodeGob(25), gobHeaders())
	})
}

var cp = completerProtocol{baseURL: "http://test.com"}

func TestCompletedValueReqWithSuccess(t *testing.T) {
	req := cp.completedValueReq("tid", "foo")
	assert.Equal(t, SuccessHeaderValue, req.Header.Get(ResultStatusHeader))
}

func TestCompletedValueReqWithError(t *testing.T) {
	req := cp.completedValueReq("tid", errors.New("foo"))
	assert.Equal(t, FailureHeaderValue, req.Header.Get(ResultStatusHeader))
}

func gobHeaders() *textproto.MIMEHeader {
	h := textproto.MIMEHeader{}
	h.Add(ContentTypeHeader, GobMediaHeader)
	h.Add(DatumTypeHeader, BlobDatumHeader)
	return &h
}

func withOneArg(one string) {
}

func withTwoArgs(one string, two int) {
}

func withThreeArgs(one string, two int, e error) {
}
