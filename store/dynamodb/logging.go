package dynamodb

import "github.com/go-kit/kit/log"

type loggingService struct {
	service
	logger log.Logger
}

func newLoggingService(logger log.Logger, s service) service {
	return &loggingService{service: s, logger: logger}
}
