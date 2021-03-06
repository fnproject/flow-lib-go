// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/validate"
)

// ModelGetGraphStateResponseStages model get graph state response stages
// swagger:model modelGetGraphStateResponseStages
type ModelGetGraphStateResponseStages map[string]GetGraphStateResponseStageRepresentation

// Validate validates this model get graph state response stages
func (m ModelGetGraphStateResponseStages) Validate(formats strfmt.Registry) error {
	var res []error

	if err := validate.Required("", "body", ModelGetGraphStateResponseStages(m)); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
