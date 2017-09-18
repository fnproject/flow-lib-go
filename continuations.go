package completions

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

var cMutex = &sync.Mutex{} // guards access to continuations
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
	cMutex.Lock()
	cMutex.Unlock()
	return continuationKey(strconv.Itoa(len(continuations)))
}

func RegisterContinuation(continuation interface{}) {
	if reflect.TypeOf(continuation).Kind() != reflect.Func {
		panic("Continuation must be a function!")
	}
	key := newContinuationKey(continuation)
	cMutex.Lock()
	defer cMutex.Unlock()
	continuations[key] = continuation
}

func invokeFromRegistry(cKey string, args ...interface{}) (interface{}, error) {
	cMutex.Lock()
	defer cMutex.Unlock()
	if e, ok := continuations[continuationKey(cKey)]; !ok {
		panic("Continuation not registered")
	} else {
		return invoke(e, args...)
	}
}

func handleContinuation(codec codec) {
	cType, ok := codec.getHeader(ContentTypeHeader)
	if !ok {
		panic("Missing content type header")
	}
	mediaType, params, err := mime.ParseMediaType(cType)
	if err != nil {
		panic("Failed to get content type for continuation")
	}
	var decoded []interface{}
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(os.Stdin, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err != nil {
				break
			}
			var val interface{}
			if len(decoded) == 0 {
				log("Unmarshalling continuation")
				val = decodeContinuation(p)
			} else {
				log(fmt.Sprintf("Unmarshalling arg %d", len(decoded)))
				val = decodeArg(decoded[0], len(decoded)-1, p)
			}
			decoded = append(decoded, val)
		}
	}

	if len(decoded) < 1 {
		panic("Invalid multipart continuation")
	}

	result, err := invoke(decoded[0], decoded[1:]...)
	writeContinuationResponse(result, err)
}

func decodeContinuation(reader io.Reader) interface{} {
	var ref continuationRef
	if err := json.NewDecoder(reader).Decode(&ref); err != nil {
		panic("Failed to decode continuation")
	}
	cMutex.Lock()
	defer cMutex.Unlock()
	function, valid := continuations[ref.Key]
	if !valid {
		panic("Continuation not registered")
	}
	return function
}
