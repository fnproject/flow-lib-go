// Code generated by go-swagger; DO NOT EDIT.

package flow_service

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"
	"time"

	"golang.org/x/net/context"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"

	strfmt "github.com/go-openapi/strfmt"

	"github.com/fnproject/flow-lib-go/models"
)

// NewAddStageParams creates a new AddStageParams object
// with the default values initialized.
func NewAddStageParams() *AddStageParams {
	var ()
	return &AddStageParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewAddStageParamsWithTimeout creates a new AddStageParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewAddStageParamsWithTimeout(timeout time.Duration) *AddStageParams {
	var ()
	return &AddStageParams{

		timeout: timeout,
	}
}

// NewAddStageParamsWithContext creates a new AddStageParams object
// with the default values initialized, and the ability to set a context for a request
func NewAddStageParamsWithContext(ctx context.Context) *AddStageParams {
	var ()
	return &AddStageParams{

		Context: ctx,
	}
}

// NewAddStageParamsWithHTTPClient creates a new AddStageParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewAddStageParamsWithHTTPClient(client *http.Client) *AddStageParams {
	var ()
	return &AddStageParams{
		HTTPClient: client,
	}
}

/*AddStageParams contains all the parameters to send to the API endpoint
for the add stage operation typically these are written to a http.Request
*/
type AddStageParams struct {

	/*Body*/
	Body *models.ModelAddStageRequest
	/*FlowID*/
	FlowID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the add stage params
func (o *AddStageParams) WithTimeout(timeout time.Duration) *AddStageParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the add stage params
func (o *AddStageParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the add stage params
func (o *AddStageParams) WithContext(ctx context.Context) *AddStageParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the add stage params
func (o *AddStageParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the add stage params
func (o *AddStageParams) WithHTTPClient(client *http.Client) *AddStageParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the add stage params
func (o *AddStageParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the add stage params
func (o *AddStageParams) WithBody(body *models.ModelAddStageRequest) *AddStageParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the add stage params
func (o *AddStageParams) SetBody(body *models.ModelAddStageRequest) {
	o.Body = body
}

// WithFlowID adds the flowID to the add stage params
func (o *AddStageParams) WithFlowID(flowID string) *AddStageParams {
	o.SetFlowID(flowID)
	return o
}

// SetFlowID adds the flowId to the add stage params
func (o *AddStageParams) SetFlowID(flowID string) {
	o.FlowID = flowID
}

// WriteToRequest writes these params to a swagger request
func (o *AddStageParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	// path param flow_id
	if err := r.SetPathParam("flow_id", o.FlowID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
