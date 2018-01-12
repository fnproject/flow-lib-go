// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
)

// ModelGraphCreatedEvent Graph created
// swagger:model modelGraphCreatedEvent
type ModelGraphCreatedEvent struct {

	// flow id
	FlowID string `json:"flow_id,omitempty"`

	// function id
	FunctionID string `json:"function_id,omitempty"`

	// ts
	Ts strfmt.DateTime `json:"ts,omitempty"`
}

// Validate validates this model graph created event
func (m *ModelGraphCreatedEvent) Validate(formats strfmt.Registry) error {
	var res []error

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// MarshalBinary interface implementation
func (m *ModelGraphCreatedEvent) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *ModelGraphCreatedEvent) UnmarshalBinary(b []byte) error {
	var res ModelGraphCreatedEvent
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
