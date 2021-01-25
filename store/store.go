/**
 * Copyright 2020 Comcast Cable Communications Management, LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

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

const DefaultMaxTTLSeconds = 60 * 24 * 365

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
