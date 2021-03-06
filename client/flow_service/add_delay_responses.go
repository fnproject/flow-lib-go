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

// AddDelayReader is a Reader for the AddDelay structure.
type AddDelayReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *AddDelayReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewAddDelayOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewAddDelayOK creates a AddDelayOK with default headers values
func NewAddDelayOK() *AddDelayOK {
	return &AddDelayOK{}
}

/*AddDelayOK handles this case with default header values.

AddDelayOK add delay o k
*/
type AddDelayOK struct {
	Payload *models.ModelAddStageResponse
}

func (o *AddDelayOK) Error() string {
	return fmt.Sprintf("[POST /v1/flows/{flow_id}/delay][%d] addDelayOK  %+v", 200, o.Payload)
}

func (o *AddDelayOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ModelAddStageResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
