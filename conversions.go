package flow

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	"github.com/fnproject/flow-lib-go/blobstore"
	"github.com/fnproject/flow-lib-go/models"
)

func closureToModel(closure interface{}, flowID string, blobStore blobstore.BlobStoreClient) *models.ModelBlobDatum {
	b := blobStore.WriteBlob(flowID, JSONMediaHeader, encodeActionRef(closure))
	debug(fmt.Sprintf("Published blob %v", b.BlobId))
	return &models.ModelBlobDatum{BlobID: b.BlobId, ContentType: b.ContentType, Length: b.BlobLength}
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

func encodeActionRef(fn interface{}) *bytes.Buffer {
	cr := newActionRef(fn)
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
