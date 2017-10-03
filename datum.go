package completions

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/textproto"
	"os"
	"reflect"
	"strconv"
)

type datum interface {
	Encode(val interface{}) bool
	DecodeArg(reflect.Type, io.Reader, *textproto.MIMEHeader) (interface{}, bool)
}

func encodeResponse(val interface{}) {
	debug(fmt.Sprintf("Encoding response of go type %v", reflect.TypeOf(val)))
	for _, t := range datumTypes {
		if t.Encode(val) {
			debug(fmt.Sprintf("Encoding result with datum type %v", reflect.TypeOf(t)))
			return
		}
	}
	panic("Failed to find suitable datum type to encode response")
}

func decodeArg(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) interface{} {
	debug(fmt.Sprintf("Decoding arg of datum type %s", header.Get(DatumTypeHeader)))
	for _, t := range datumTypes {
		if res, ok := t.DecodeArg(argType, reader, header); ok {
			debug(fmt.Sprintf("Decoded result with datum type %v", reflect.TypeOf(t)))
			return res
		}
	}
	panic("Failed to find suitable datum type to decode arg")
}

// datum types in order of priority
var datumTypes = []datum{new(emptyDatum), new(errorDatum), new(stageDatum), new(blobDatum)}

type emptyDatum struct{}

func (d *emptyDatum) Encode(val interface{}) bool {
	if val != nil {
		return false
	}
	fmt.Printf("HTTP/1.1 200\r\n")
	fmt.Printf("%s: %s\r\n", DatumTypeHeader, EmptyDatumHeader)
	fmt.Printf("%s: %s\r\n", ResultStatusHeader, SuccessHeaderValue)
	fmt.Printf("\r\n")
	return true
}

func (d *emptyDatum) DecodeArg(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) (interface{}, bool) {
	if header.Get(DatumTypeHeader) != EmptyDatumHeader {
		return nil, false
	}
	return nil, true
}

type blobDatum struct{}

func (d *blobDatum) Encode(val interface{}) bool {
	buf := encodeGob(val)
	fmt.Printf("HTTP/1.1 200\r\n")
	fmt.Printf("%s: %s\r\n", ContentTypeHeader, GobMediaHeader)
	fmt.Printf("Content-Length: %d\r\n", buf.Len())
	fmt.Printf("%s: %s\r\n", DatumTypeHeader, BlobDatumHeader)
	fmt.Printf("%s: %s\r\n", ResultStatusHeader, SuccessHeaderValue)
	fmt.Printf("\r\n")
	buf.WriteTo(os.Stdout)
	return true
}

func (d *blobDatum) DecodeArg(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) (interface{}, bool) {
	if header.Get(DatumTypeHeader) != BlobDatumHeader {
		return nil, false
	}
	return decodeBlob(argType, reader, header), true
}

type errorDatum struct{}

func (d *errorDatum) Encode(val interface{}) bool {
	if e, ok := val.(error); ok {
		errMsg := e.Error()
		buf := encodeGob(&errMsg)
		fmt.Printf("HTTP/1.1 200\r\n")
		fmt.Printf("%s: %s\r\n", ContentTypeHeader, GobMediaHeader)
		fmt.Printf("Content-Length: %d\r\n", buf.Len())
		fmt.Printf("%s: %s\r\n", DatumTypeHeader, BlobDatumHeader)
		fmt.Printf("%s: %s\r\n", ResultStatusHeader, FailureHeaderValue)
		fmt.Printf("\r\n")
		buf.WriteTo(os.Stdout)
		return true
	}
	return false
}

func (d *errorDatum) DecodeArg(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) (interface{}, bool) {
	if header.Get(DatumTypeHeader) != ErrorDatumHeader {
		return nil, false
	}
	errType := header.Get(ErrorTypeHeader)
	debug(fmt.Sprintf("Processing error of type %s", errType))
	errMsg := "Unknown error details"
	if readBytes, readError := ioutil.ReadAll(reader); readError == nil {
		errMsg = string(readBytes)
	}
	if errType != "" {
		errMsg = fmt.Sprintf("%s: %s", errType, errMsg)
	}
	return errors.New(errMsg), true
}

type stageDatum struct{}

func (d *stageDatum) Encode(val interface{}) bool {
	if cf, ok := val.(*cloudFuture); ok {
		debug(fmt.Sprintf("Returning stage ref %s", cf.completionID))
		fmt.Printf("HTTP/1.1 200\r\n")
		fmt.Printf("%s: %s\r\n", DatumTypeHeader, StageRefDatumHeader)
		fmt.Printf("%s: %s\r\n", ResultStatusHeader, SuccessHeaderValue)
		fmt.Printf("%s: %s\r\n", StageIDHeader, cf.completionID)
		fmt.Printf("\r\n")
		return true
	}
	return false
}

func (d *stageDatum) DecodeArg(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) (interface{}, bool) {
	if header.Get(DatumTypeHeader) != StageRefDatumHeader {
		return nil, false
	}
	stageID := header.Get(StageIDHeader)
	return cloudFuture{
		cloudThread:  CurrentThread().(*cloudThread),
		completionID: completionID(stageID),
	}, true
}

type httpReqDatum struct{}

func (d *httpReqDatum) Encode(val interface{}) bool {
	return false
}

func (d *httpReqDatum) DecodeArg(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) (interface{}, bool) {
	if header.Get(DatumTypeHeader) != HTTPReqDatumHeader {
		return nil, false
	}
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
	}, true
}

type httpRespDatum struct{}

func (d *httpRespDatum) Encode(val interface{}) bool {
	return false
}

func (d *httpRespDatum) DecodeArg(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) (interface{}, bool) {
	if header.Get(DatumTypeHeader) != HTTPRespDatumHeader {
		return nil, false
	}
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
	}, true
}
