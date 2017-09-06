package completions

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"time"
)

const (
	// protocol headers
	HeaderPrefix       = "FnProject-"
	DatumTypeHeader    = HeaderPrefix + "Datumtype"
	ThreadIDHeader     = HeaderPrefix + "ThreadID"
	StageIDHeader      = HeaderPrefix + "StageID"
	ResultStatusHeader = HeaderPrefix + "ResultStatus"

	SuccessHeaderValue = "success"
	FailureHeaderValue = "failure"

	BlobDatumHeader = "blob"

	// standard headers
	ContentTypeHeader = "Content-Type"
	GobMediaHeader    = "application/x-gob"

	MaxContinuationArgCount = 2
)

type completerProtocol struct {
	baseURL string
}

func newCompleterProtocol(baseURL string) *completerProtocol {
	return &completerProtocol{baseURL: baseURL}
}

type continuationRef struct {
	Key continuationKey `json:"continuation-key"`
}

func newContinuationRef(function interface{}) *continuationRef {
	return &continuationRef{Key: newContinuationKey(function)}
}

func (p *completerProtocol) parseThreadID(res *http.Response) threadID {
	return threadID(res.Header.Get(ThreadIDHeader))
}

func (p *completerProtocol) parseStageID(res *http.Response) completionID {
	return completionID(res.Header.Get(StageIDHeader))
}

func (p *completerProtocol) createThreadReq(functionID string) *http.Request {
	url := fmt.Sprintf("%s/graph?functionId=%s", p.baseURL, functionID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		panic("Failed to create request object")
	}
	return req
}

func (p *completerProtocol) completedValueReq(tid threadID, value interface{}) *http.Request {
	req := createRequest("POST", fmt.Sprintf("%s/graph/%s/completedValue", p.baseURL, tid), encodeGob(value))
	req.Header.Set(DatumTypeHeader, BlobDatumHeader)
	req.Header.Set(ContentTypeHeader, GobMediaHeader)
	return req
}

func (p *completerProtocol) delayReq(tid threadID, duration time.Duration) *http.Request {
	return createRequest("POST", fmt.Sprintf("%s/graph/%s/delay?delayMs=%d", p.baseURL, tid, int64(duration)), nil)
}

func (p *completerProtocol) thenApplyReq(tid threadID, cid completionID, function interface{}) *http.Request {
	ref := newContinuationRef(function)
	b, err := json.Marshal(ref)
	if err != nil {
		panic("Failed to marshal continuation reference")
	}
	req := createRequest("POST", fmt.Sprintf("%s/graph/%s/stage/%s/thenApply", p.baseURL, tid, cid), bytes.NewReader(b))
	req.Header.Set(DatumTypeHeader, BlobDatumHeader)
	req.Header.Set(ContentTypeHeader, "application/json")
	return req
}

func (p *completerProtocol) getStageReq(tid threadID, cid completionID) *http.Request {
	return createRequest("GET", fmt.Sprintf("%s/graph/%s/stage/%s", p.baseURL, tid, cid), nil)
}

func (p *completerProtocol) commit(tid threadID) *http.Request {
	return createRequest("POST", fmt.Sprintf("%s/graph/%s/commit", p.baseURL, tid), nil)
}

// panics if the request can't be created
func createRequest(method string, url string, r io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		panic("Failed to create request object")
	}
	return req
}

func encodeGob(value interface{}) *bytes.Buffer {
	var buf bytes.Buffer
	buf.Len()
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		panic("Failed to encode gob: " + err.Error())
	}
	return &buf
}

func decodeGob(r io.Reader, val interface{}) {
	dec := gob.NewDecoder(r)
	if err := dec.Decode(val); err != nil {
		panic("Failed to decode gob: " + err.Error())
	}
}

func decodeTypedGob(r io.Reader, t reflect.Type) interface{} {
	dec := gob.NewDecoder(r)
	v := reflect.New(t)
	ref := v.Interface()
	if err := dec.Decode(ref); err != nil {
		panic("Failed to decode gob: " + err.Error())
	}
	return ref
}

func continuationArgTypes(continuation interface{}) (argTypes []reflect.Type) {
	if reflect.TypeOf(continuation).Kind() != reflect.Func {
		panic("Continuation must be a function!")
	}

	fn := reflect.TypeOf(continuation)
	argC := fn.NumIn() // inbound params
	if argC > MaxContinuationArgCount {
		panic(fmt.Sprintf("Continuations may take a maximum of %d parameters", MaxContinuationArgCount))
	}
	argTypes = make([]reflect.Type, argC)
	for i := 0; i < argC; i++ {
		argTypes[i] = fn.In(i)
	}
	return
}

func decodeContinuationArgs(continuation interface{}, inputs ...io.Reader) (results []interface{}) {
	argTypes := continuationArgTypes(continuation)
	results = make([]interface{}, len(argTypes))
	if len(argTypes) != len(inputs) {
		panic("Invalid number of arguments decoded for continuation")
	}
	for i, input := range inputs {
		// TODO depending on the header decode as gob
		results[i] = decodeTypedGob(input, argTypes[i])
	}
	return
}

func writeContinuationResponse(result interface{}, err error) {
	fmt.Printf("HTTP/1.1 200\r\n")
	fmt.Printf("%s: %s\r\n", ContentTypeHeader, GobMediaHeader)

	var buf *bytes.Buffer
	var status string
	if err != nil {
		buf = encodeGob(err)
		status = FailureHeaderValue
	} else {
		buf = encodeGob(result)
		status = SuccessHeaderValue
	}
	fmt.Printf("Content-Length: %d\r\n", buf.Len())
	fmt.Printf("%s: blob\r\n", DatumTypeHeader)
	fmt.Printf("%s: %s\r\n", ResultStatusHeader, status)
	fmt.Printf("\r\n")

	buf.WriteTo(os.Stdout)
}
