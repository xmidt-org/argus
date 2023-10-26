// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/xmidt-org/touchstone"
	"go.uber.org/fx"
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
func ProvideMetrics() fx.Option {
	return fx.Options(
		touchstone.CounterVec(
			prometheus.CounterOpts{
				Name: PollCounter,
				Help: "Counter for the number of polls (and their success/failure outcomes) to fetch new items.",
			},
			OutcomeLabel,
		),
	)
}

type Measures struct {
	fx.In
	Polls *prometheus.CounterVec `name:"chrysom_polls_total"`
}
