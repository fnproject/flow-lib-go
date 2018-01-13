package flows

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

var actionsMtx = &sync.Mutex{} // guards access to cActions
var actions = make(map[string]interface{})

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

func getActionID(action interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(action).Pointer()).Name()
}

// registers a go function so it can be used as an action
// in a flow stage
func RegisterAction(action interface{}) {
	if reflect.TypeOf(action).Kind() != reflect.Func {
		panic("Action must be a function!")
	}
	actionsMtx.Lock()
	defer actionsMtx.Unlock()
	actions[getActionID(action)] = action
}

func invokeFromRegistry(actionID string, args ...interface{}) (interface{}, error) {
	actionsMtx.Lock()
	defer actionsMtx.Unlock()
	if e, ok := actions[actionID]; !ok {
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
		mr := multipart.NewReader(os.Stdin, params["boundary"])
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
		encodeDatum(codec.out(), err)
	} else {
		encodeDatum(codec.out(), result)
	}
}

func decodeContinuation(reader io.Reader) interface{} {
	var ref continuationRef
	if err := json.NewDecoder(reader).Decode(&ref); err != nil {
		panic("Failed to decode continuation")
	}
	actionsMtx.Lock()
	defer actionsMtx.Unlock()
	action, valid := actions[ref.ID]
	if !valid {
		panic("Continuation not registered")
	}
	return action
}
