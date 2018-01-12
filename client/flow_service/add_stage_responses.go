// Code generated by go-swagger; DO NOT EDIT.

package flow_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/fnproject/flow-lib-go/models"
)

// AddStageReader is a Reader for the AddStage structure.
type AddStageReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *AddStageReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewAddStageOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewAddStageOK creates a AddStageOK with default headers values
func NewAddStageOK() *AddStageOK {
	return &AddStageOK{}
}

/*AddStageOK handles this case with default header values.

AddStageOK add stage o k
*/
type AddStageOK struct {
	Payload *models.ModelAddStageResponse
}

func (o *AddStageOK) Error() string {
	return fmt.Sprintf("[POST /v1/flows/{flow_id}/stage][%d] addStageOK  %+v", 200, o.Payload)
}

func (o *AddStageOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ModelAddStageResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
