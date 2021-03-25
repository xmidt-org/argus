package store

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/httpaux"
)

// Sentinel internal errors.
var (
	ErrItemNotFound   = errors.New("Item at resource path not found")
	ErrBucketNotFound = errors.New("Bucket path not found")
	ErrJSONDecode     = errors.New("Error decoding JSON data from DB")
	ErrJSONEncode     = errors.New("Error encoding JSON data to send to DB")
	ErrQueryExecution = errors.New("Error occurred during DB query execution")
)

// Sentinel errors to be used by the HTTP response error encoder.
var (
	ErrHTTPItemNotFound   = httpaux.Error{Err: errors.New("Item not found"), Code: http.StatusNotFound}
	ErrHTTPBucketNotFound = httpaux.Error{Err: errors.New("Bucket not found"), Code: http.StatusNotFound}
	ErrHTTPOpFailed       = httpaux.Error{Err: errors.New("DB operation failed"), Code: http.StatusInternalServerError}
)

type sanitizedErrorer interface {
	// SanitizedError returns an error message with content
	// that should be safe to share across API boundaries.
	// This provides a mechanism to prevent leaking
	// sensitive error data coming from the datastores.
	SanitizedError() string
}

type SanitizedError struct {
	// Err contains the raw error explaining the cause of the
	// failure event.
	Err error

	// ErrHTTP should contain some filtered version
	// of Err that can be safely used across API boundaries.
	// Two use cases include: hiding sensitive error data and
	// translating errors to better explain error events to API consumers.
	ErrHTTP httpaux.Error
}

func (s SanitizedError) Unwrap() error {
	return s.Err
}

func (s SanitizedError) Error() string {
	return s.Err.Error()
}

func (s SanitizedError) SanitizedError() string {
	return s.ErrHTTP.Error()
}

func (s SanitizedError) StatusCode() int {
	return s.ErrHTTP.StatusCode()
}

func (s SanitizedError) Headers() http.Header {
	return s.ErrHTTP.Headers()
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

// ItemOperationError is a simple error wrapper for DB operations
// that apply to specific items. It provides a formatted message with
// context around the error.
type ItemOperationError struct {
	Err       error
	Key       model.Key
	Operation string
}

func (e ItemOperationError) Error() string {
	return fmt.Sprintf("%s operation failed for item at path %s/%s: %v", e.Operation, e.Key.Bucket, e.Key.ID, e.Err)
}

func (e ItemOperationError) Unwrap() error {
	return e.Err
}

// GetAllItemsOperationError is the ItemOperation counterpart for
// the getAllItems operation which applies to a group of items.
type GetAllItemsOperationErr struct {
	Err    error
	Bucket string
}

func (e GetAllItemsOperationErr) Error() string {
	return fmt.Sprintf("getall operation failed for bucket %s: %v", e.Bucket, e.Err)
}

func (e GetAllItemsOperationErr) Unwrap() error {
	return e.Err
}

// SanitizeError should be used by DB implementations to prevent exposing
// internal error data in HTTP responses.
// This method maps an internal error to their sanitized version which contains
// HTTP response information like code and debug header values.
// DB implementations should implement their own versions of this function
// when they need to look at implementation-specific errors to perform the mapping.
func SanitizeError(err error) error {
	if err == nil {
		return nil
	}
	var errHTTP = ErrHTTPOpFailed

	switch {
	case errors.Is(err, ErrItemNotFound):
		errHTTP = ErrHTTPItemNotFound
	case errors.Is(err, ErrBucketNotFound):
		errHTTP = ErrHTTPBucketNotFound
	}
	return SanitizedError{Err: err, ErrHTTP: errHTTP}
}
