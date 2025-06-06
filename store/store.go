// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"github.com/xmidt-org/argus/model"
)

const (
	// TypeLabel is for labeling metrics; if there is a single metric for
	// successful queries, the typeLabel and corresponding type can be used
	// when incrementing the metric.
	TypeLabel  = "type"
	InsertType = "insert"
	DeleteType = "delete"
	ReadType   = "read"
	PingType   = "ping"
)

type S interface {
	Push(key model.Key, item OwnableItem) error
	Get(key model.Key) (OwnableItem, error)
	Delete(key model.Key) (OwnableItem, error)
	GetAll(bucket string) (map[string]OwnableItem, error)
}

type OwnableItem struct {
	model.Item
	Owner string `json:"owner"`
}

func FilterOwner(value map[string]OwnableItem, owner string) map[string]OwnableItem {
	filteredResults := map[string]OwnableItem{}
	for k, v := range value {
		if v.Owner == owner {
			filteredResults[k] = v
		}
	}
	return filteredResults
}
