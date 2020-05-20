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

// 5 minutes
const DefaultTTL = 60 * 5

type S interface {
	Push(key model.Key, item InternalItem) error
	Get(key model.Key) (InternalItem, error)
	Delete(key model.Key) (InternalItem, error)
	GetAll(bucket string) (map[string]InternalItem, error)
}

type InternalItem struct {
	model.Item

	Owner string
}

func FilterOwner(value map[string]InternalItem, owner string) map[string]InternalItem {
	if owner == "" {
		return value
	}

	filteredResults := map[string]InternalItem{}
	for k, v := range value {
		if v.Owner == owner {
			filteredResults[k] = v
		}
	}
	return filteredResults
}
