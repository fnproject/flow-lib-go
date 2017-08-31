package completions

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// protocol headers
	HeaderPrefix    = "FnProject-"
	DatumTypeHeader = HeaderPrefix + "Datumtype"
	ThreadIDHeader  = HeaderPrefix + "ThreadID"
	StageIDHeader   = HeaderPrefix + "StageID"

	BlobDatumHeader = "blob"

	// standard headers
	ContentTypeHeader = "Content-Type"
	GobMediaHeader    = "application/x-gob"
)

type completerProtocol struct {
	baseURL string
}

func newCompleterProtocol(baseURL string) *completerProtocol {
	return &completerProtocol{baseURL: baseURL}
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
	return createRequest("POST", fmt.Sprintf("%s/graph/%s/completedValue", p.baseURL, tid), encodeGob(value))
}

func (p *completerProtocol) delayReq(tid threadID, duration time.Duration) *http.Request {
	return createRequest("POST", fmt.Sprintf("%s/graph/%s/delay?delayMs=%d", p.baseURL, tid, int64(duration)), nil)
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
