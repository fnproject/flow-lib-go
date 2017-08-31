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

func (p *completerProtocol) completedValueReq(threadID threadID, value interface{}) *http.Request {
	url := fmt.Sprintf("%s/graph/%s/completedValue", p.baseURL, threadID)
	req, err := http.NewRequest("POST", url, encodeGob(value))
	req.Header.Set(DatumTypeHeader, BlobDatumHeader)
	req.Header.Set(ContentTypeHeader, GobMediaHeader)
	if err != nil {
		panic("Failed to create request object")
	}
	return req
}

func (p *completerProtocol) delayReq(threadID threadID, duration time.Duration) *http.Request {
	url := fmt.Sprintf("%s/graph/%s/delay?delayMs=%d", p.baseURL, threadID, int64(duration))
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		panic("Failed to create request object")
	}
	return req
}

func (p *completerProtocol) getStageReq(threadID threadID, completionID completionID) *http.Request {
	url := fmt.Sprintf("%s/graph/%s/stage/%s", p.baseURL, threadID, completionID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic("Failed to create request object")
	}
	return req
}

func encodeGob(value interface{}) *bytes.Buffer {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		panic("Failed to encode gob: " + err.Error())
	}
	return &buf
}

func decodeGob(r io.Reader) interface{} {
	dec := gob.NewDecoder(r)
	var decoded interface{}
	if err := dec.Decode(decoded); err != nil {
		panic("Failed to decode gob: " + err.Error())
	}
	return decoded
}
