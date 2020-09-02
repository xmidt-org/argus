package dynamodb

import "github.com/xmidt-org/argus/store/db/metric"

type instrumentingService struct {
	service
	measures metric.Measures
}

func newInstrumentingService(measures metric.Measures, s service) service {
	return &instrumentingService{measures: measures, service: s}
}
