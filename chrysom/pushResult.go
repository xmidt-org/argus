// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package chrysom

// PushResult is a simple type to indicate the result type for the
// PushItem operation.
type PushResult int64

// Types of pushItem successful results.
const (
	UnknownPushResult PushResult = iota
	CreatedPushResult
	UpdatedPushResult
	NilPushResult
)

func (p PushResult) String() string {
	switch p {
	case NilPushResult:
		return ""
	case CreatedPushResult:
		return "created"
	case UpdatedPushResult:
		return "ok"
	}
	return "unknown"
}
