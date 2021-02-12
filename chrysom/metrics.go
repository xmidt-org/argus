package chrysom

import (
	"github.com/xmidt-org/webpa-common/xmetrics"
)

// Names
const (
	PollCounter = "chrysom_polls_total"
)

// Labels
const (
	OutcomeLabel = "outcome"
)

// Label Values
const (
	SuccessOutcome = "success"
	FailureOutcome = "failure"
)

// Metrics returns the Metrics relevant to this package
func Metrics() []xmetrics.Metric {
	return []xmetrics.Metric{
		{
			Name:       PollCounter,
			Type:       xmetrics.CounterType,
			Help:       "Counter for the number of polls (and their success/failure outcomes) to fetch new items.",
			LabelNames: []string{OutcomeLabel},
		},
	}
}
