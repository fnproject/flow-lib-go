package completions

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"reflect"
	"strconv"
	"strings"
)

const (
	// protocol headers
	HeaderPrefix       = "FnProject-"
	DatumTypeHeader    = HeaderPrefix + "Datumtype"
	ThreadIDHeader     = HeaderPrefix + "FlowID"
	StageIDHeader      = HeaderPrefix + "StageID"
	ResultStatusHeader = HeaderPrefix + "ResultStatus"
	ResultCodeHeader   = HeaderPrefix + "ResultCode"
	CodeLocationHeader = HeaderPrefix + "Codeloc"
	ErrorTypeHeader    = HeaderPrefix + "Errortype"
	MethodHeader       = HeaderPrefix + "Method"

	UserHeaderPrefix = HeaderPrefix + "Header-"

	SuccessHeaderValue = "success"
	FailureHeaderValue = "failure"

	BlobDatumHeader     = "blob"
	EmptyDatumHeader    = "empty"
	ErrorDatumHeader    = "error"
	StageRefDatumHeader = "stageref"
	HTTPReqDatumHeader  = "httpreq"
	HTTPRespDatumHeader = "httpresp"
	StateDatumHeader    = "state"

	// standard headers
	ContentTypeHeader        = "Content-Type"
	JSONMediaHeader          = "application/json"
	GobMediaHeader           = "application/x-gob"
	TextMediaHeader          = "text/plain"
	OctetStreamMediaHeader   = "application/octet-stream"
	DefaultContentTypeHeader = OctetStreamMediaHeader

	MaxContinuationArgCount = 2
)

type completerProtocol struct {
	baseURL string
}

func newCompleterProtocol(baseURL string) *completerProtocol {
	return &completerProtocol{baseURL: baseURL}
}

type continuationRef struct {
	Key cKey `json:"continuation-key"`
}

func (cr *continuationRef) getKey() string {
	return string(cr.Key)
}

func newContinuationRef(function interface{}) *continuationRef {
	return &continuationRef{Key: continuationKey(function)}
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
	URL := p.rootStageURL("completedValue", tid)
	var req *http.Request
	if err, isErr := value.(error); isErr {
		req = createRequest("POST", URL, strings.NewReader(err.Error()))
		req.Header.Set(ResultStatusHeader, FailureHeaderValue)
		req.Header.Set(ErrorTypeHeader, "user-defined-error")
		req.Header.Set(DatumTypeHeader, ErrorDatumHeader)
		req.Header.Set(ContentTypeHeader, TextMediaHeader)
	} else {
		req = createRequest("POST", URL, encodeGob(value))
		req.Header.Set(ResultStatusHeader, SuccessHeaderValue)
		req.Header.Set(DatumTypeHeader, BlobDatumHeader)
		req.Header.Set(ContentTypeHeader, GobMediaHeader)
	}
	return req
}

func (p *completerProtocol) rootStageURL(op string, tid threadID) string {
	return fmt.Sprintf("%s/graph/%s/%s", p.baseURL, tid, op)
}

func (p *completerProtocol) chainedStageURL(op string, tid threadID, cid completionID) string {
	return fmt.Sprintf("%s/graph/%s/stage/%s/%s", p.baseURL, tid, cid, op)
}

func (p *completerProtocol) chained(op string, tid threadID, cid completionID, fn interface{}, loc *codeLoc) *http.Request {
	return p.completionWithBody(p.chainedStageURL(op, tid, cid), fn, loc)
}

func (p *completerProtocol) chainedWithOther(op string, tid threadID, cid completionID, altCid completionID, fn interface{}, loc *codeLoc) *http.Request {
	URL := fmt.Sprintf("%s/graph/%s/stage/%s/%s?other=%s", p.baseURL, tid, cid, op, string(altCid))
	return p.completionWithBody(URL, fn, loc)
}

func (p *completerProtocol) completionWithBody(URL string, fn interface{}, loc *codeLoc) *http.Request {
	b, err := json.Marshal(newContinuationRef(fn))
	if err != nil {
		panic("Failed to marshal continuation reference")
	}
	return p.completion(URL, loc, bytes.NewReader(b))
}

func (p *completerProtocol) invokeFunction(URL string, loc *codeLoc, r HTTPRequest) *http.Request {
	req := createRequest("POST", URL, bytes.NewReader(r.Body))
	req.Header.Set(DatumTypeHeader, HTTPReqDatumHeader)
	req.Header.Set(MethodHeader, r.Method)
	cType := r.Headers.Get(ContentTypeHeader)
	if cType == "" {
		cType = DefaultContentTypeHeader
	}
	req.Header.Set(ContentTypeHeader, cType)
	req.Header.Set(CodeLocationHeader, loc.String())
	for k, v := range r.Headers {
		// don't allow duplicate values for the same key
		req.Header.Set(UserHeaderPrefix+k, v[0])
	}
	return req
}

func (p *completerProtocol) completion(URL string, loc *codeLoc, r io.Reader) *http.Request {
	req := createRequest("POST", URL, r)
	req.Header.Set(ContentTypeHeader, JSONMediaHeader)
	req.Header.Set(DatumTypeHeader, BlobDatumHeader)
	req.Header.Set(CodeLocationHeader, loc.String())
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
	debug(fmt.Sprintf("Requesting URL %s", url))
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
	var v reflect.Value
	if t.Kind() == reflect.Ptr {
		v = reflect.New(t.Elem())
	} else {
		v = reflect.New(t)
	}
	if err := dec.Decode(v.Interface()); err != nil {
		panic("Failed to decode gob: " + err.Error())
	}

	if t.Kind() == reflect.Ptr {
		return v.Interface()
	}
	return v.Elem().Interface()
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

func decodeArg(continuation interface{}, argIndex int, reader io.Reader, header *textproto.MIMEHeader) interface{} {
	argTypes := continuationArgTypes(continuation)
	if len(argTypes) < argIndex {
		panic("Invalid number of arguments decoded for continuation")
	}
	datumType := header.Get(DatumTypeHeader)
	debug(fmt.Sprintf("Decoding arg of type %s", datumType))
	switch datumType {
	case BlobDatumHeader:
		return decodeBlob(argTypes[argIndex], reader, header)
	case EmptyDatumHeader:
		return nil
	case ErrorDatumHeader:
		errType := header.Get(ErrorTypeHeader)
		debug(fmt.Sprintf("Processing error of type %s", errType))
		errMsg := "Unknown error details"
		if readBytes, readError := ioutil.ReadAll(reader); readError == nil {
			errMsg = string(readBytes)
		}
		if errType != "" {
			errMsg = fmt.Sprintf("%s: %s", errType, errMsg)
		}
		return errors.New(errMsg)

	case StageRefDatumHeader:
		stageID := header.Get(StageIDHeader)
		return cloudFuture{completionID: completionID(stageID)}

	case HTTPReqDatumHeader:
		method := header.Get(MethodHeader)
		if method == "" {
			method = http.MethodPost
		}

		body, bodyErr := ioutil.ReadAll(reader)
		if bodyErr == nil {
			panic("Failed to read body of HTTP response")
		}

		headers := http.Header{}
		for k, values := range *header {
			for _, v := range values {
				headers.Set(k, v)
			}
		}

		return HTTPRequest{
			Method:  method,
			Body:    body,
			Headers: headers,
		}

	case HTTPRespDatumHeader:
		code := header.Get(ResultCodeHeader)
		statusCode, statusErr := strconv.Atoi(code)
		if statusErr != nil {
			panic("Invalid result code for HTTP response: " + code)
		}

		body, bodyErr := ioutil.ReadAll(reader)
		if bodyErr == nil {
			panic("Failed to read body of HTTP response")
		}

		headers := http.Header{}
		for k, values := range *header {
			for _, v := range values {
				headers.Set(k, v)
			}
		}
		return HTTPResponse{
			StatusCode: statusCode,
			Body:       body,
			Headers:    headers,
		}

	default:
		panic("Unkown content type in http multipart")
	}
}

func decodeBlob(t reflect.Type, reader io.Reader, header *textproto.MIMEHeader) interface{} {
	switch header.Get(ContentTypeHeader) {
	case GobMediaHeader:
		return decodeTypedGob(reader, t)
	default:
		panic("Unkown content type for blob")
	}
}

func writeContinuationResponse(result interface{}) {
	encodeVal(result)
}
