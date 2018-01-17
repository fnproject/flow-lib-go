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

// models incoming request API (not auto-generated from swagger!)
type StageInvocation struct {
	FlowID  string                          `json:"flow_id,omitempty"`
	StageID string                          `json:"stage_id,omitempty"`
	Closure *models.ModelDatum              `json:"closure,omitempty"`
	Args    []*models.ModelCompletionResult `json:"args,omitempty"`
}

func (in *StageInvocation) invoke() {
	// catch panics and publish them as errors
	defer func() {
		if r := recover(); r != nil {
			stack := fmt.Sprintf("%s: %s", r, dbg.Stack()) // line 20
			debug(fmt.Sprintf("Recovered from invoke error:\n %s", stack))
			// TODO complete with error
		}
	}()

	debug(fmt.Sprintf("Invoking continuation with %d args", len(in.Args)))

	if in.Closure == nil || in.Closure.Blob == nil {
		panic("Got empty closure blob while invoking stage")
	}

	var actionFunc interface{}
	slurp := func(body io.ReadCloser) {
		var ref continuationRef
		if err := json.NewDecoder(body).Decode(&ref); err != nil {
			panic("Failed to decode continuation")
		}

		actionsMtx.Lock()
		defer actionsMtx.Unlock()
		var valid bool
		actionFunc, valid = actions[ref.ID]
		if !valid {
			panic("Continuation not registered")
		}
	}

	blobstore.GetBlobStore().ReadBlob(in.FlowID, in.Closure.Blob.BlobID, JSONMediaHeader, slurp)

	argTypes := continuationArgTypes(actionFunc)
	var args []interface{}
	for i, arg := range in.Args {
		debug(fmt.Sprintf("Decoding arg of type %v", argTypes[i]))
		args = append(args, arg.DecodeValue(in.FlowID, argTypes[i], blobstore.GetBlobStore()))
	}

	result, err := invokeFunc(actionFunc, args)
	debug(fmt.Sprintf("Continuation returned %d values", len(args)))
	if err != nil {
		// TODO publish error
		debug(fmt.Sprintf("Got err %v", err))
		// writeError to Stdout
		return
	}
	// TODO publish result
	debug(fmt.Sprintf("Got result %v", result))
	// writeResult to Stdout
}

func invokeFunc(continuation interface{}, args []interface{}) (result interface{}, err error) {
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
	var in StageInvocation
	if err := json.NewDecoder(os.Stdin).Decode(&in); err != nil {
		panic("Failed to decode continuation")
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
		datum.StageRef = &models.ModelStageRefDatum{StageID: string(v.stageID)}

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
	b := blobStore.WriteBlob(flowID, GobMediaHeader, encodeContinuationRef(closure))
	return &models.ModelBlobDatum{BlobID: b.BlobId, ContentType: b.ContentType, Length: b.BlobLength}
}
