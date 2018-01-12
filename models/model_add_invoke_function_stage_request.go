// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
)

// ModelAddInvokeFunctionStageRequest AddInvokeFunctionStageRequest adds a function invocation
// swagger:model modelAddInvokeFunctionStageRequest
type ModelAddInvokeFunctionStageRequest struct {

	// arg
	Arg *ModelHTTPReqDatum `json:"arg,omitempty"`

	// caller id
	CallerID string `json:"caller_id,omitempty"`

	// code location
	CodeLocation string `json:"code_location,omitempty"`

	// flow id
	FlowID string `json:"flow_id,omitempty"`

	// function id
	FunctionID string `json:"function_id,omitempty"`
}

// Validate validates this model add invoke function stage request
func (m *ModelAddInvokeFunctionStageRequest) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateArg(formats); err != nil {
		// prop
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *ModelAddInvokeFunctionStageRequest) validateArg(formats strfmt.Registry) error {

	if swag.IsZero(m.Arg) { // not required
		return nil
	}

	if m.Arg != nil {

		if err := m.Arg.Validate(formats); err != nil {
			if ve, ok := err.(*errors.Validation); ok {
				return ve.ValidateName("arg")
			}
			return err
		}
	}

	return nil
}

// MarshalBinary interface implementation
func (m *ModelAddInvokeFunctionStageRequest) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ModelAddInvokeFunctionStageRequest) UnmarshalBinary(b []byte) error {
	var res ModelAddInvokeFunctionStageRequest
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
