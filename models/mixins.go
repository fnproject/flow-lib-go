package models

import (
	"reflect"
)

type Decoder interface {
	DecodeResult(t reflect.Type) interface{}
	DecodeError() error
}

func (result *ModelCompletionResult) DecodeValue(t reflect.Type) interface{} {
	if result.Successful {
		return result.Datum.DecodeResult(t)
	}
	return result.Datum.DecodeError()
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

func (d *ModelDatum) DecodeResult(t reflect.Type) interface{} {
	if d.Empty != nil {
		return nil
	}
	return d.decoder().DecodeResult(t)
}

func (d *ModelDatum) DecodeError() error {
	if d.Empty != nil {
		return nil
	}
	return d.decoder().DecodeError()
}

func (d *ModelBlobDatum) DecodeResult(t reflect.Type) interface{} {

	return nil
}

func (d *ModelBlobDatum) DecodeError() error {
	return nil
}

func (d *ModelErrorDatum) DecodeResult(t reflect.Type) interface{} {
	return nil
}

func (d *ModelErrorDatum) DecodeError() error {
	return nil
}

func (d *ModelHTTPReqDatum) DecodeError() error {
	return nil
}

func (d *ModelHTTPReqDatum) DecodeResult(t reflect.Type) interface{} {
	return nil
}

func (d *ModelHTTPRespDatum) DecodeError() error {
	return nil
}

func (d *ModelHTTPRespDatum) DecodeResult(t reflect.Type) interface{} {
	return nil
}

func (d *ModelStageRefDatum) DecodeError() error {
	return nil
}

func (d *ModelStageRefDatum) DecodeResult(t reflect.Type) interface{} {
	return nil
}

func (d *ModelStatusDatum) DecodeError() error {
	return nil
}

func (d *ModelStatusDatum) DecodeResult(t reflect.Type) interface{} {
	return nil
}
