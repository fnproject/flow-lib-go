package completions

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"reflect"
	"strings"
)

var continuations = make(map[continuationKey]interface{})

func invoke(continuation interface{}, args ...interface{}) (result interface{}, err error) {
	// catch panics and return them as errors
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	values := invokeContinuation(continuation, args...)
	switch len(values) {
	case 0:
		return nil, nil
	case 1:
		return valToInterface(values[0]), nil
	case 2:
		return valToInterface(values[0]), valToError(values[1])
	default:
		return nil, fmt.Errorf("Invalid continuation")
	}
}

func valToInterface(v reflect.Value) interface{} {
	return v.Interface()
}

func valToError(v reflect.Value) error {
	if v.IsNil() {
		return nil
	}
	return valToInterface(v).(error)
}

func invokeContinuation(continuation interface{}, args ...interface{}) []reflect.Value {
	v := reflect.ValueOf(continuation)
	rargs := make([]reflect.Value, len(args))
	for i, a := range args {
		rargs[i] = reflect.ValueOf(a)
	}
	return v.Call(rargs)
}

type continuationKey string

func newContinuationKey(function interface{}) continuationKey {
	return continuationKey(reflect.TypeOf(function).String())
}

func RegisterContinuation(continuation interface{}) {
	if reflect.TypeOf(continuation).Kind() != reflect.Func {
		panic("Continuation must be a function!")
	}
	continuations[newContinuationKey(continuation)] = continuation
}

func invokeFromRegistry(cKey string, args ...interface{}) (interface{}, error) {
	if e, ok := continuations[continuationKey(cKey)]; !ok {
		panic("Continuation not registered")
	} else {
		return invoke(e, args...)
	}
}

func handleContinuation(codec codec) {
	mediaType, params, err := mime.ParseMediaType(codec.getHeader(ContentTypeHeader))
	if err != nil {
		panic("Failed to get content type for continuation")
	}
	var parts []*multipart.Part
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(os.Stdin, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				return
			}
			if err != nil {
				panic("Failed to parse multipart continuation")
			}
			parts = append(parts, p)
			//slurp, err := ioutil.ReadAll(p)
		}
	}

	if len(parts) < 1 {
		panic("Invalid multipart continuation")
	}

	//slurp, err := ioutil.ReadAll(parts[0])
	var ref continuationRef
	if err := json.NewDecoder(parts[0]).Decode(&ref); err != nil {
		panic("Failed to decode continuation")
	}

	function, valid := continuations[ref.Key]
	if !valid {
		panic("Continuation not registered")
	}

	var bodies []io.Reader
	for i := 1; i < len(parts); i++ {
		bodies = append(bodies, parts[i])
	}
	args := decodeContinuationArgs(function, bodies...)
	result, error := invoke(ref.Key, args)
	writeContinuationResponse(result, err)
}
