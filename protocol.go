package completions

import (
	"bytes"
	"encoding/gob"
	"fmt"
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

type completerPotocol struct {
	baseURL string
}

func (p *completerPotocol) parseThreadID(res *http.Response) threadID {
	return threadID(res.Header.Get(ThreadIDHeader))
}

func (p *completerPotocol) parseStageID(res *http.Response) completionID {
	return completionID(res.Header.Get(StageIDHeader))
}

func (p *completerPotocol) createThreadReq(functionID string) *http.Request {
	url := fmt.Sprintf("%s/graph?functionId=%s", p.baseURL, functionID)
	req, err := http.NewRequest("POST", url, &bytes.Buffer{})
	if err != nil {
		panic("Failed to create request object")
	}
	return req
}

func (p *completerPotocol) completedValueReq(threadID threadID, value interface{}) *http.Request {
	url := fmt.Sprintf("%s/graph/%s/completedValue", p.baseURL, threadID)
	req, err := http.NewRequest("POST", url, encodeGob(value))
	req.Header.Set(DatumTypeHeader, BlobDatumHeader)
	req.Header.Set(ContentTypeHeader, GobMediaHeader)
	if err != nil {
		panic("Failed to create request object")
	}
	return req
}

func (p *completerPotocol) delayReq(threadID threadID, duration time.Duration) *http.Request {
	url := fmt.Sprintf("%s/graph/%s/delay?delayMs=%d", p.baseURL, threadID, int64(duration))
	req, err := http.NewRequest("POST", url, &bytes.Buffer{})
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
