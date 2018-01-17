package models

import (
	"encoding/gob"
	"fmt"
	"io"
	"reflect"

	"github.com/fnproject/flow-lib-go/blobstore"
)

const (
	GobMediaHeader = "application/x-gob"
)

type Decoder interface {
	DecodeResult(fid string, t reflect.Type, blobStore blobstore.BlobStoreClient) interface{}
	DecodeError(fid string, blobStore blobstore.BlobStoreClient) error
}

func (result *ModelCompletionResult) DecodeValue(fid string, t reflect.Type, blobStore blobstore.BlobStoreClient) interface{} {
	if result.Successful {
		return result.Datum.DecodeResult(fid, t, blobStore)
	}
	return result.Datum.DecodeError(fid, blobStore)
}

func (d *ModelDatum) decoder() Decoder {
	if d.Blob != nil {
		return d.Blob
	} else if d.Error != nil {
		return d.Error
	} else if d.HTTPReq != nil {
		return d.HTTPReq
	} else if d.HTTPResp != nil {
		return d.HTTPResp
	} else if d.StageRef != nil {
		return d.StageRef
	} else if d.Status != nil {
		return d.Status
	}
	panic("Received empty datum!")
}

func (d *ModelDatum) DecodeResult(fid string, t reflect.Type, blobStore blobstore.BlobStoreClient) interface{} {
	if d.Empty != nil {
		return nil
	}
	return d.decoder().DecodeResult(fid, t, blobStore)
}

func (d *ModelDatum) DecodeError(fid string, blobStore blobstore.BlobStoreClient) error {
	if d.Empty != nil {
		return nil
	}
	return d.decoder().DecodeError(fid, blobStore)
}

func (d *ModelBlobDatum) DecodeResult(fid string, t reflect.Type, blobStore blobstore.BlobStoreClient) (result interface{}) {
	if d.ContentType != GobMediaHeader {
		panic(fmt.Sprintf("Unsupported blob content type %v", d.ContentType))
	}
	blobStore.ReadBlob(fid, d.BlobID, d.ContentType, func(body io.ReadCloser) { result = decodeGob(body, t) })
	return
}

func (d *ModelBlobDatum) DecodeError(fid string, blobStore blobstore.BlobStoreClient) error {
	return nil
}

func (d *ModelErrorDatum) DecodeResult(fid string, t reflect.Type, blobStore blobstore.BlobStoreClient) interface{} {
	return nil
}

func (d *ModelErrorDatum) DecodeError(fid string, blobStore blobstore.BlobStoreClient) error {
	return nil
}

func (d *ModelHTTPReqDatum) DecodeError(fid string, blobStore blobstore.BlobStoreClient) error {
	return nil
}

func (d *ModelHTTPReqDatum) DecodeResult(fid string, t reflect.Type, blobStore blobstore.BlobStoreClient) interface{} {
	return nil
}

func (d *ModelHTTPRespDatum) DecodeError(fid string, blobStore blobstore.BlobStoreClient) error {
	return nil
}

func (d *ModelHTTPRespDatum) DecodeResult(fid string, t reflect.Type, blobStore blobstore.BlobStoreClient) interface{} {
	return nil
}

func (d *ModelStageRefDatum) DecodeError(fid string, blobStore blobstore.BlobStoreClient) error {
	return nil
}

func (d *ModelStageRefDatum) DecodeResult(fid string, t reflect.Type, blobStore blobstore.BlobStoreClient) interface{} {
	return nil
}

func (d *ModelStatusDatum) DecodeError(fid string, blobStore blobstore.BlobStoreClient) error {
	return nil
}

func (d *ModelStatusDatum) DecodeResult(fid string, t reflect.Type, blobStore blobstore.BlobStoreClient) interface{} {
	return nil
}

func decodeGob(r io.Reader, t reflect.Type) interface{} {
	if t == nil {
		// use GetType(reflect.Type)
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
