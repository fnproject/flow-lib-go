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

// CreateGraphReader is a Reader for the CreateGraph structure.
type CreateGraphReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *CreateGraphReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewCreateGraphOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewCreateGraphOK creates a CreateGraphOK with default headers values
func NewCreateGraphOK() *CreateGraphOK {
	return &CreateGraphOK{}
}

/*CreateGraphOK handles this case with default header values.

CreateGraphOK create graph o k
*/
type CreateGraphOK struct {
	Payload *models.ModelCreateGraphResponse
}

func (o *CreateGraphOK) Error() string {
	return fmt.Sprintf("[POST /v1/flows][%d] createGraphOK  %+v", 200, o.Payload)
}

func (o *CreateGraphOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ModelCreateGraphResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
