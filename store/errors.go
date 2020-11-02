package store

import (
	"fmt"
	"net/http"

	"github.com/xmidt-org/argus/model"
)

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
	if knfe.Key.UUID == "" && knfe.Key.Bucket == "" {
		return fmt.Sprint("parameters for key not set")
	} else if knfe.Key.UUID == "" && knfe.Key.Bucket != "" {
		return fmt.Sprintf("no value exists for bucket %s", knfe.Key.Bucket)

	}
	return fmt.Sprintf("no value exists with bucket: %s, uuid: %s", knfe.Key.Bucket, knfe.Key.UUID)
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
