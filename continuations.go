package completions

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"reflect"
	"runtime"
	dbg "runtime/debug"
	"strings"
	"sync"
)

var cMutex = &sync.Mutex{} // guards access to cFunctions
var cFunctions = make(map[cKey]interface{})

func invoke(continuation interface{}, args ...interface{}) (result interface{}, err error) {
	// catch panics and return them as errors
	defer func() {
		if r := recover(); r != nil {
			stack := fmt.Sprintf("%s: %s", r, dbg.Stack()) // line 20
			debug(fmt.Sprintf("Recovered from invoke error:\n %s", stack))
			err = fmt.Errorf("%v", r)
		}
	}()
	debug(fmt.Sprintf("Invoking continuation with %d args", len(args)))
	values := invokeContinuation(continuation, args...)
	debug(fmt.Sprintf("Continuation returned %d values", len(args)))
	switch len(values) {
	case 0:
		return nil, nil
	case 1:
		return valToInterface(values[0]), nil
	case 2:
		return valToInterface(values[0]), valToError(values[1])
	default:
		return nil, fmt.Errorf("Continuation returned invalid number of values")
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
	fn := reflect.ValueOf(continuation)
	var rargs []reflect.Value
	argTypes := continuationArgTypes(continuation)

	if reflect.TypeOf(continuation).NumIn() == 0 {
		debug("Ignoring arguments for empty continuation function")
		rargs = make([]reflect.Value, 0)
	} else {
		rargs = make([]reflect.Value, len(args))
		for i, a := range args {
			if a == nil { // converts empty datum parameters to zero type
				rargs[i] = reflect.Zero(argTypes[i])
			} else {
				rargs[i] = reflect.ValueOf(a)
			}
		}
	}
	return fn.Call(rargs)
}

type cKey string

func continuationKey(c interface{}) cKey {
	return cKey(runtime.FuncForPC(reflect.ValueOf(c).Pointer()).Name())
}

func RegisterContinuation(function interface{}) {
	if reflect.TypeOf(function).Kind() != reflect.Func {
		panic("Continuation must be a function!")
	}
	cMutex.Lock()
	defer cMutex.Unlock()
	cFunctions[continuationKey(function)] = function
}

func invokeFromRegistry(key string, args ...interface{}) (interface{}, error) {
	cMutex.Lock()
	defer cMutex.Unlock()
	if e, ok := cFunctions[cKey(key)]; !ok {
		panic("Continuation not registered")
	} else {
		return invoke(e, args...)
	}
}

func handleContinuation(codec codec) {
	debug("Handling continuation")
	cType, ok := codec.getHeader(ContentTypeHeader)
	if !ok {
		panic("Missing content type header")
	}
	debug(fmt.Sprintf("Handling continuation of type %s", cType))
	mediaType, params, err := mime.ParseMediaType(cType)
	if err != nil {
		panic("Failed to get content type for continuation")
	}
	var decoded []interface{}
	if strings.HasPrefix(mediaType, "multipart/") {
		// stickyErrorReader workaround for https://github.com/golang/go/issues/14676
		mr := multipart.NewReader(&stickyErrorReader{r: os.Stdin}, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err != nil {
				break
			}
			var val interface{}
			if len(decoded) == 0 {
				debug("Unmarshalling continuation")
				val = decodeContinuation(p)
			} else {
				debug(fmt.Sprintf("Unmarshalling arg %d", len(decoded)))
				val = decodeContinuationArg(decoded[0], len(decoded)-1, p, &p.Header)
			}
			decoded = append(decoded, val)
		}
	}

	if len(decoded) < 1 {
		panic("Invalid multipart continuation")
	}

	result, err := invoke(decoded[0], decoded[1:]...)
	// stages can only receive one value for a completion
	if err != nil {
		encodeDatum(err)
	} else {
		encodeDatum(result)
	}
}

func decodeContinuation(reader io.Reader) interface{} {
	var ref continuationRef
	if err := json.NewDecoder(reader).Decode(&ref); err != nil {
		panic("Failed to decode continuation")
	}
	cMutex.Lock()
	defer cMutex.Unlock()
	function, valid := cFunctions[ref.Key]
	if !valid {
		panic("Continuation not registered")
	}
	return function
}

// from https://go-review.googlesource.com/c/go/+/22045/3/src/mime/multipart/multipart.go#99
// stickyErrorReader is an io.Reader which never calls Read on its
// underlying Reader once an error has been seen. (the io.Reader
// interface's contract promises nothing about the return values of
// Read calls after an error, yet this package does do multiple Reads
// after error)
type stickyErrorReader struct {
	r   io.Reader
	err error
}

func (r *stickyErrorReader) Read(p []byte) (n int, _ error) {
	if r.err != nil {
		return 0, r.err
	}
	n, r.err = r.r.Read(p)
	return n, r.err
}
