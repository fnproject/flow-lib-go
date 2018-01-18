package flow

import (
	"bytes"
	"encoding/gob"
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
	HeaderPrefix  = "FnProject-"
	FlowIDHeader  = HeaderPrefix + "FlowID"
	StageIDHeader = HeaderPrefix + "StageID"

	JSONMediaHeader         = "application/json"
	GobMediaHeader          = "application/x-gob"
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
	for i, arg := range in.Args {
		debug(fmt.Sprintf("Decoding arg of type %v", argTypes[i]))
		args = append(args, arg.DecodeValue(in.FlowID, argTypes[i], blobstore.GetBlobStore()))
	}

	result, err := invokeFunc(actionFunc, args)
	writeResult(in.FlowID, result, err)
}

func (in *InvokeStageRequest) action() (actionFunction interface{}) {
	blobstore.GetBlobStore().ReadBlob(in.FlowID, in.Closure.BlobID, JSONMediaHeader,
		func(body io.ReadCloser) {
			var ref continuationRef
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

type continuationRef struct {
	ID string `json:"action-id"`
}

func (cr *continuationRef) getKey() string {
	return cr.ID
}

func newContinuationRef(action interface{}) *continuationRef {
	return &continuationRef{ID: getActionID(action)}
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

func valueToModel(value interface{}, flowID string, blobStore blobstore.BlobStoreClient) *models.ModelCompletionResult {
	datum := new(models.ModelDatum)
	switch v := value.(type) {
	case *flowFuture:
		datum.StageRef = &models.ModelStageRefDatum{StageID: v.stageID}

	default:
		if value == nil {
			datum.Empty = new(models.ModelEmptyDatum)
		} else {
			b := blobStore.WriteBlob(flowID, GobMediaHeader, encodeGob(value))
			datum.Blob = &models.ModelBlobDatum{BlobID: b.BlobId, ContentType: b.ContentType, Length: b.BlobLength}
		}
	}

	_, isErr := value.(error)
	return &models.ModelCompletionResult{Successful: !isErr, Datum: datum}
}

func closureToModel(closure interface{}, flowID string, blobStore blobstore.BlobStoreClient) *models.ModelBlobDatum {
	b := blobStore.WriteBlob(flowID, JSONMediaHeader, encodeContinuationRef(closure))
	debug(fmt.Sprintf("Published blob %v", b.BlobId))
	return &models.ModelBlobDatum{BlobID: b.BlobId, ContentType: b.ContentType, Length: b.BlobLength}
}

func actionArgs(continuation interface{}) (argTypes []reflect.Type) {
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

func encodeContinuationRef(fn interface{}) *bytes.Buffer {
	cr := newContinuationRef(fn)
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(cr); err != nil {
		panic("Failed to encode continuation reference: " + err.Error())
	}
	return &buf
}

func encodeGob(value interface{}) *bytes.Buffer {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(value); err != nil {
		panic("Failed to encode gob: " + err.Error())
	}
	return &buf
}
