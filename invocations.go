package flow

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	dbg "runtime/debug"

	"github.com/fnproject/flow-lib-go/blobstore"
	"github.com/fnproject/flow-lib-go/models"
)

const (
	// protocol headers
	HeaderPrefix      = "FnProject-"
	FlowIDHeader      = HeaderPrefix + "FlowID"
	ContentTypeHeader = "Content-Type"

	JSONMediaHeader         = "application/json"
	GobMediaHeader          = "application/x-gob"
	OctetStreamMediaHeader  = "application/octet-stream"
	MaxContinuationArgCount = 2
)

// models incoming request API (not auto-generated from swagger!)
type InvokeStageRequest struct {
	FlowID  string                          `json:"flow_id,omitempty"`
	StageID string                          `json:"stage_id,omitempty"`
	Closure *models.ModelBlobDatum          `json:"closure,omitempty"`
	Args    []*models.ModelCompletionResult `json:"args,omitempty"`
}

type InvokeStageResponse struct {
	Result *models.ModelCompletionResult `json:"result,omitempty"`
}

func (in *InvokeStageRequest) invoke() {
	// catch panics and publish them as errors
	defer func() {
		if r := recover(); r != nil {
			stack := fmt.Sprintf("%s: %s", r, dbg.Stack())
			debug(fmt.Sprintf("Recovered from invoke error:\n %s", stack))
		}
	}()

	debug(fmt.Sprintf("Invoking continuation with %d args", len(in.Args)))

	actionFunc := in.action()
	argTypes := actionArgs(actionFunc)

	var args []interface{}
	for i, _ := range argTypes {
		debug(fmt.Sprintf("Decoding arg of type %v", argTypes[i]))
		args = append(args, decodeResult(in.Args[i], in.FlowID, argTypes[i], blobstore.GetBlobStore()))
	}

	result, err := invokeFunc(actionFunc, args)
	writeResult(in.FlowID, result, err)
}

func (in *InvokeStageRequest) action() (actionFunction interface{}) {
	blobstore.GetBlobStore().ReadBlob(in.FlowID, in.Closure.BlobID, JSONMediaHeader,
		func(body io.ReadCloser) {
			var ref actionRef
			if err := json.NewDecoder(body).Decode(&ref); err != nil {
				panic("Failed to decode continuation")
			}

			var valid bool
			actionFunction, valid = actions[ref.ID]
			if !valid {
				panic("Continuation not registered")
			}
		})
	return
}

func handleInvocation(codec codec) {
	debug("Handling continuation")
	var in InvokeStageRequest
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		panic(fmt.Sprintf("Failed to decode stage invocation request: %v", err))
	}
	if len(in.Args) < 1 {
		panic("Invalid multipart continuation, need at least one argument")
	}

	in.invoke()
}

func invokeFunc(continuation interface{}, args []interface{}) (result interface{}, err error) {
	fn := reflect.ValueOf(continuation)
	var rargs []reflect.Value
	argTypes := actionArgs(continuation)

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

	results := fn.Call(rargs)
	switch len(results) {
	case 0:
		return nil, nil
	case 1:
		return valToInterface(results[0]), nil
	case 2:
		return valToInterface(results[0]), valToError(results[1])
	default:
		return nil, fmt.Errorf("Continuation returned invalid number of results")
	}
}

func writeResult(flowID string, result interface{}, err error) {
	var val interface{}
	if err == nil {
		debug(fmt.Sprintf("Writing successful result %v", result))
		val = result
	} else {
		debug(fmt.Sprintf("Writing error result %v", err))
		val = err
	}
	resp := &InvokeStageResponse{Result: valueToModel(val, flowID, blobstore.GetBlobStore())}
	if err := json.NewEncoder(os.Stdout).Encode(resp); err != nil {
		panic("Failed to encode completion result")
	}
}

// internal encoding of a function pointer since go doesn't allow pointers to be serialized
type actionRef struct {
	ID string `json:"action-key"`
}

func (cr *actionRef) getKey() string {
	return cr.ID
}

func newActionRef(actionFunc interface{}) *actionRef {
	return &actionRef{ID: getActionKey(actionFunc)}
}

func actionArgs(actionFunc interface{}) (argTypes []reflect.Type) {
	if reflect.TypeOf(actionFunc).Kind() != reflect.Func {
		panic("Continuation must be a function!")
	}

	fn := reflect.TypeOf(actionFunc)
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

func valToInterface(v reflect.Value) interface{} {
	return v.Interface()
}

func valToError(v reflect.Value) error {
	if v.IsNil() {
		return nil
	}
	return valToInterface(v).(error)
}
