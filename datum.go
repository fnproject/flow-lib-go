package completions

import (
	"fmt"
	"io"
	"os"
	"reflect"
)

type datum interface {
	Encode(val interface{}) bool
	//Decode(io.Reader) (interface{}, bool)
}

func encodeVal(val interface{}) {
	for _, t := range datumTypes {
		if t.Encode(val) {
			debug(fmt.Sprintf("Encoding result of type %v", reflect.TypeOf(t)))
			return
		}
	}
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

func (d *emptyDatum) Decode(reader io.Reader) (interface{}, bool) {
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
