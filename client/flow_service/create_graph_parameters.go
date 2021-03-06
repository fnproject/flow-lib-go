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

// NewCreateGraphParams creates a new CreateGraphParams object
// with the default values initialized.
func NewCreateGraphParams() *CreateGraphParams {
	var ()
	return &CreateGraphParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewCreateGraphParamsWithTimeout creates a new CreateGraphParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewCreateGraphParamsWithTimeout(timeout time.Duration) *CreateGraphParams {
	var ()
	return &CreateGraphParams{

		timeout: timeout,
	}
}

// NewCreateGraphParamsWithContext creates a new CreateGraphParams object
// with the default values initialized, and the ability to set a context for a request
func NewCreateGraphParamsWithContext(ctx context.Context) *CreateGraphParams {
	var ()
	return &CreateGraphParams{

		Context: ctx,
	}
}

// NewCreateGraphParamsWithHTTPClient creates a new CreateGraphParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewCreateGraphParamsWithHTTPClient(client *http.Client) *CreateGraphParams {
	var ()
	return &CreateGraphParams{
		HTTPClient: client,
	}
}

/*CreateGraphParams contains all the parameters to send to the API endpoint
for the create graph operation typically these are written to a http.Request
*/
type CreateGraphParams struct {

	/*Body*/
	Body *models.ModelCreateGraphRequest

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the create graph params
func (o *CreateGraphParams) WithTimeout(timeout time.Duration) *CreateGraphParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the create graph params
func (o *CreateGraphParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the create graph params
func (o *CreateGraphParams) WithContext(ctx context.Context) *CreateGraphParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the create graph params
func (o *CreateGraphParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the create graph params
func (o *CreateGraphParams) WithHTTPClient(client *http.Client) *CreateGraphParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the create graph params
func (o *CreateGraphParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithBody adds the body to the create graph params
func (o *CreateGraphParams) WithBody(body *models.ModelCreateGraphRequest) *CreateGraphParams {
	o.SetBody(body)
	return o
}

// SetBody adds the body to the create graph params
func (o *CreateGraphParams) SetBody(body *models.ModelCreateGraphRequest) {
	o.Body = body
}

// WriteToRequest writes these params to a swagger request
func (o *CreateGraphParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	if o.Body != nil {
		if err := r.SetBodyParam(o.Body); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
