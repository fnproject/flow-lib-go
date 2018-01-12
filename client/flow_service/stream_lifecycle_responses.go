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

// StreamLifecycleReader is a Reader for the StreamLifecycle structure.
type StreamLifecycleReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *StreamLifecycleReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewStreamLifecycleOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewStreamLifecycleOK creates a StreamLifecycleOK with default headers values
func NewStreamLifecycleOK() *StreamLifecycleOK {
	return &StreamLifecycleOK{}
}

/*StreamLifecycleOK handles this case with default header values.

(streaming responses)
*/
type StreamLifecycleOK struct {
	Payload *models.ModelGraphLifecycleEvent
}

func (o *StreamLifecycleOK) Error() string {
	return fmt.Sprintf("[GET /v1/stream][%d] streamLifecycleOK  %+v", 200, o.Payload)
}

func (o *StreamLifecycleOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.ModelGraphLifecycleEvent)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
