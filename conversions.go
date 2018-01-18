package flow

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"

	"github.com/fnproject/flow-lib-go/blobstore"
	"github.com/fnproject/flow-lib-go/models"
)

func actionToModel(actionFunc interface{}, flowID string, blobStore blobstore.BlobStoreClient) *models.ModelBlobDatum {
	b := blobStore.WriteBlob(flowID, JSONMediaHeader, encodeAction(actionFunc))
	debug(fmt.Sprintf("Published blob %v", b.BlobId))
	return &models.ModelBlobDatum{BlobID: b.BlobId, ContentType: b.ContentType, Length: b.BlobLength}
}

func valueToModel(value interface{}, flowID string, blobStore blobstore.BlobStoreClient) *models.ModelCompletionResult {
	datum := new(models.ModelDatum)
	switch v := value.(type) {

	case FlowFuture:
		debug("Converting value to ModelStageRefDatum")
		ff, ok := v.(*flowFuture)
		if !ok {
			panic("Third-party implementations of FlowFuture are not supported!")
		}
		datum.StageRef = &models.ModelStageRefDatum{StageID: ff.stageID}

	case *models.ModelHTTPReqDatum:
		debug("Converting value to ModelHTTPReqDatum")
		datum.HTTPReq = v

	case *models.ModelHTTPRespDatum:
		debug("Converting value to ModelHTTPRespDatum")
		datum.HTTPResp = v

	default:
		if value == nil {
			debug("Converting value to ModelEmptyDatum")
			datum.Empty = new(models.ModelEmptyDatum)
			break
		}

		var body io.Reader
		var contentType string
		if errv, isErr := value.(error); isErr {
			body = encodeError(errv)
			contentType = JSONMediaHeader
		} else {
			body = encodeGob(value)
			contentType = GobMediaHeader
		}
		debug("Converting value to ModelBlobDatum")
		b := blobStore.WriteBlob(flowID, contentType, body)
		datum.Blob = &models.ModelBlobDatum{BlobID: b.BlobId, ContentType: b.ContentType, Length: b.BlobLength}
	}

	_, isErr := value.(error)
	return &models.ModelCompletionResult{Successful: !isErr, Datum: datum}
}

func encodeAction(actionFunc interface{}) *bytes.Buffer {
	cr := newActionRef(actionFunc)
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(cr); err != nil {
		log.Fatalf("Failed to encode continuation reference: %v", err.Error())
	}
	return &buf
}

func encodeGob(value interface{}) *bytes.Buffer {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(value); err != nil {
		log.Fatalf("Failed to encode gob: %v", err.Error())
	}
	return &buf
}

func encodeError(e error) *bytes.Buffer {
	result := &ErrorResult{Error: e.Error()}
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(result); err != nil {
		log.Fatalf("Failed to encode error: %v", err.Error())
	}
	return &buf
}

// converts back to Go and API types - yuck!
func decodeResult(result *models.ModelCompletionResult, flowID string, rType reflect.Type, blobStore blobstore.BlobStoreClient) interface{} {
	datum := result.Datum.InnerDatum()
	if result.Successful {
		return datumToValue(datum, flowID, rType, blobStore)
	} else {
		return datumToError(datum, flowID, blobStore)
	}
}

func datumToValue(datum interface{}, flowID string, rType reflect.Type, blobStore blobstore.BlobStoreClient) interface{} {
	switch d := datum.(type) {
	case *models.ModelEmptyDatum:
		return nil

	case *models.ModelBlobDatum:
		if d.ContentType != GobMediaHeader {
			panic(fmt.Sprintf("Unsupported blob content type %v", d.ContentType))
		}
		var result interface{}
		blobStore.ReadBlob(flowID, d.BlobID, d.ContentType, func(b io.ReadCloser) { result = decodeGob(b, rType) })
		return result

	case *models.ModelHTTPReqDatum:
		// TODO convert to HTTPReq
		return &HTTPRequest{}

	case *models.ModelHTTPRespDatum:
		// TODO convert to HTTPResp
		return &HTTPResponse{}

	case *models.ModelStageRefDatum:
		return &flowFuture{flow: CurrentFlow().(*flow), stageID: d.StageID}

	case *models.ModelStatusDatum:
		// TODO turn this into an iota in the API
		return fmt.Sprintf("%v", d.Type)

	default:
		panic(fmt.Sprintf("Successful result %v cannot be decoded to go type", reflect.TypeOf(datum)))
	}
}

func datumToError(datum interface{}, flowID string, blobStore blobstore.BlobStoreClient) error {
	switch d := datum.(type) {
	case *models.ModelBlobDatum:
		if d.ContentType != JSONMediaHeader {
			panic(fmt.Sprintf("Unsupported blob content type for error %v", d.ContentType))
		}
		var err error
		blobStore.ReadBlob(flowID, d.BlobID, d.ContentType, func(b io.ReadCloser) { err = decodeError(b) })
		return err

	case *models.ModelErrorDatum:
		return errors.New(fmt.Sprintf("Platform error %v: %v", d.Type, d.Message))

	default:
		panic(fmt.Sprintf("Failure result %v cannot be decoded to go type", reflect.TypeOf(datum)))
	}
}

// errors cannot be encoded using gobs, so we just extract the message and encode with json
type ErrorResult struct {
	Error string `json:"error"`
}

func (e *ErrorResult) Err() error {
	return errors.New(e.Error)
}

func decodeError(r io.Reader) error {
	result := new(ErrorResult)
	if err := json.NewDecoder(r).Decode(result); err != nil {
		panic("Failed to decode error result")
	}
	return result.Err()
}

func decodeGob(r io.Reader, t reflect.Type) interface{} {
	if t == nil {
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
