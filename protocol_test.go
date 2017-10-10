package flows

import (
	"bytes"
	"errors"
	"fmt"
	"net/textproto"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeAndDecodeGob(t *testing.T) {
	buf := encodeGob(25)
	var result int
	result = decodeGob(buf, reflect.TypeOf(result)).(int)
	assert.Equal(t, 25, result)
}

func TestEncodeAndDecodeGobInterface(t *testing.T) {
	buf := encodeGob(25)
	result := decodeGob(buf, reflect.TypeOf(25))
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
	req := cp.completedValueReq("fid", "foo")
	assert.Equal(t, SuccessHeaderValue, req.Header.Get(ResultStatusHeader))
}

func TestCompletedValueReqWithError(t *testing.T) {
	req := cp.completedValueReq("fid", errors.New("foo"))
	assert.Equal(t, FailureHeaderValue, req.Header.Get(ResultStatusHeader))
}

type Foo struct {
	Name string
}

func (f *Foo) SayHello() string {
	return "Hello " + f.Name
}

func TestMethodReceiver(t *testing.T) {
	f := &Foo{Name: "Bar"}
	b := encodeGob(f).Bytes()
	rt := encodeGob(reflect.TypeOf(f)).Bytes()
	r := &continuationRef{ID: getActionID(f.SayHello), Receiver: b, RcvType: rt}
	dec := decodeGob(bytes.NewReader(r.Receiver), reflect.TypeOf(f))
	result := reflect.ValueOf(dec).MethodByName("SayHello").Call([]reflect.Value{})
	fmt.Println(result)
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
