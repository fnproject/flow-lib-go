package flows

import (
	"encoding/gob"
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
	Decode(reflect.Type, io.Reader, *textproto.MIMEHeader) (interface{}, bool)
}

func encodeDatum(val interface{}) {
	debug(fmt.Sprintf("Encoding datum of go type %v", reflect.TypeOf(val)))
	for _, t := range datumTypes {
		if t.Encode(val) {
			debug(fmt.Sprintf("Encoding result with datum type %v", reflect.TypeOf(t)))
			return
		}
	}
	panic("Failed to find suitable datum type to encode response")
}

func decodeDatum(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) interface{} {
	debug(fmt.Sprintf("Decoding datum of type %s", header.Get(DatumTypeHeader)))
	for _, t := range datumTypes {
		if res, ok := t.Decode(argType, reader, header); ok {
			debug(fmt.Sprintf("Decoded result with datum type %v", reflect.TypeOf(t)))
			return res
		}
	}
	panic("Failed to find suitable datum type to decode arg")
}

// datum types in order of priority
var datumTypes = []datum{new(emptyDatum), new(errorDatum), new(stageDatum), new(httpReqDatum), new(httpRespDatum), new(blobDatum)}

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

func (d *emptyDatum) Decode(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) (interface{}, bool) {
	if header.Get(DatumTypeHeader) != EmptyDatumHeader {
		return nil, false
	}
	return nil, true
}

type blobDatum struct{}

func (d *blobDatum) Encode(val interface{}) bool {
	// errors are encoded as string blob datums
	if err, isErr := val.(error); isErr {
		buf := encodeGob(err.Error())
		fmt.Printf("HTTP/1.1 200\r\n")
		fmt.Printf("%s: %s\r\n", ContentTypeHeader, GobMediaHeader)
		fmt.Printf("Content-Length: %d\r\n", buf.Len())
		fmt.Printf("%s: %s\r\n", DatumTypeHeader, BlobDatumHeader)
		fmt.Printf("%s: %s\r\n", ResultStatusHeader, FailureHeaderValue)
		fmt.Printf("\r\n")
		buf.WriteTo(os.Stdout)
		return true
	}

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

func (d *blobDatum) Decode(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) (interface{}, bool) {
	if header.Get(DatumTypeHeader) != BlobDatumHeader {
		return nil, false
	}
	switch header.Get(ContentTypeHeader) {
	case GobMediaHeader:
		debug(fmt.Sprintf("Decoding gob of type %v", argType))
		// we use gobs for encoding errors as strings
		if header.Get(ResultStatusHeader) == FailureHeaderValue {
			errString := decodeGob(reader, argType).(string)
			return errors.New(errString), true
		}
		return decodeGob(reader, argType), true
	default:
		panic("Unkown content type for blob")
	}
}

type errorDatum struct{}

func (d *errorDatum) Encode(val interface{}) bool {
	// error datums are only currently generated on the server-side
	return false
}

func (d *errorDatum) Decode(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) (interface{}, bool) {
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
	if cf, ok := val.(*flowFuture); ok {
		debug(fmt.Sprintf("Returning stage ref %s", cf.stageID))
		fmt.Printf("HTTP/1.1 200\r\n")
		fmt.Printf("%s: %s\r\n", DatumTypeHeader, StageRefDatumHeader)
		fmt.Printf("%s: %s\r\n", ResultStatusHeader, SuccessHeaderValue)
		fmt.Printf("%s: %s\r\n", StageIDHeader, cf.stageID)
		fmt.Printf("\r\n")
		return true
	}
	return false
}

func (d *stageDatum) Decode(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) (interface{}, bool) {
	if header.Get(DatumTypeHeader) != StageRefDatumHeader {
		return nil, false
	}
	sid := header.Get(StageIDHeader)
	if sid == "" {
		panic("Missing stage ID header")
	}
	return flowFuture{
		flow:    CurrentFlow().(*flow),
		stageID: stageID(sid),
	}, true
}

type httpReqDatum struct{}

func (d *httpReqDatum) Encode(val interface{}) bool {
	return false
}

func (d *httpReqDatum) Decode(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) (interface{}, bool) {
	if header.Get(DatumTypeHeader) != HTTPReqDatumHeader {
		return nil, false
	}
	method := header.Get(MethodHeader)
	if method == "" {
		method = http.MethodPost
	}
	body, bodyErr := ioutil.ReadAll(reader)
	if bodyErr != nil {
		debug("Failed to read body of HTTP response: " + bodyErr.Error())
	}
	headers := http.Header{}
	for k, values := range *header {
		for _, v := range values {
			headers.Set(k, v)
		}
	}
	if header.Get(ResultStatusHeader) == FailureHeaderValue {
		return errors.New(string(body)), true
	}
	return &HTTPRequest{
		Method:  method,
		Body:    body,
		Headers: headers,
	}, true
}

type httpRespDatum struct{}

func (d *httpRespDatum) Encode(val interface{}) bool {
	return false
}

func (d *httpRespDatum) Decode(argType reflect.Type, reader io.Reader, header *textproto.MIMEHeader) (interface{}, bool) {
	if header.Get(DatumTypeHeader) != HTTPRespDatumHeader {
		return nil, false
	}
	code := header.Get(ResultCodeHeader)
	statusCode, statusErr := strconv.Atoi(code)
	if statusErr != nil {
		panic("Invalid result code for HTTP response: " + code)
	}
	body, bodyErr := ioutil.ReadAll(reader)
	if bodyErr != nil {
		debug("Failed to read body of HTTP response: " + bodyErr.Error())
	}
	headers := http.Header{}
	for k, values := range *header {
		for _, v := range values {
			headers.Set(k, v)
		}
	}
	if header.Get(ResultStatusHeader) == FailureHeaderValue {
		return errors.New(string(body)), true
	}
	return &HTTPResponse{
		StatusCode: statusCode,
		Body:       body,
		Headers:    headers,
	}, true
}

func decodeGob(r io.Reader, t reflect.Type) interface{} {
	if t == nil {
		// use GetType(reflect.Type)
		panic("Decode type could not be inferred")
	}
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
