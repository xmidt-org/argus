package store

import "net/http"

type BadRequestErr struct {
	Message string
}

func (bre BadRequestErr) Error() string {
	return bre.Message
}

func (bre BadRequestErr) StatusCode() int {
	return http.StatusBadRequest
}
