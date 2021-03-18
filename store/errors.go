package store

import (
	"fmt"
	"net/http"

	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/httpaux"
)

type sanitizedError interface {
	// Sanitized returns an HTTP error with content
	// that should be safe to share across API boundaries.
	// This provides a mechanism to prevent leaking
	// sensitive error data coming from the datastores.
	Sanitized() error
}

type SanitizedError struct {
	Err          error
	SanitizedErr httpaux.Error
}

func (s SanitizedError) Sanitized() error {
	return &s.SanitizedErr
}

func (s SanitizedError) Error() string {
	return s.Err.Error()
}

func (s SanitizedError) Unwrap() error {
	return s.Err
}

type BadRequestErr struct {
	Message string
}

func (bre BadRequestErr) Error() string {
	return bre.Message
}

func (bre BadRequestErr) StatusCode() int {
	return http.StatusBadRequest
}

type ForbiddenRequestErr struct {
	Message string
}

func (f ForbiddenRequestErr) Error() string {
	return f.Message
}

func (f ForbiddenRequestErr) StatusCode() int {
	return http.StatusForbidden
}

type KeyNotFoundError struct {
	Key model.Key
}

func (knfe KeyNotFoundError) Error() string {
	if knfe.Key.ID == "" && knfe.Key.Bucket == "" {
		return fmt.Sprint("parameters for key not set")
	} else if knfe.Key.ID == "" && knfe.Key.Bucket != "" {
		return fmt.Sprintf("no value exists for bucket %s", knfe.Key.Bucket)

	}
	return fmt.Sprintf("no value exists with bucket: %s, id: %s", knfe.Key.Bucket, knfe.Key.ID)
}

func (knfe KeyNotFoundError) StatusCode() int {
	return http.StatusNotFound
}

type InternalError struct {
	Reason    interface{}
	Retryable bool
}

func (ie InternalError) Error() string {
	return fmt.Sprintf("Request Failed: \n%#v", ie.Reason)
}

func (ie InternalError) StatusCode() int {
	return http.StatusInternalServerError
}
