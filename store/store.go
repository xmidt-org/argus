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
	Push(key Key, item Item) error
	Get(key Key) (Item, error)
	Delete(key Key) (Item, error)
	GetAll(bucket string) (map[string]Item, error)
}

type Key struct {
	Bucket string
	ID     string
}

type Attributes map[string]string

func compare(a Attributes, b Attributes) bool {
	if &a == &b {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if b[i] != v {
			return false
		}
	}
	return true
}

type Item struct {
	Attributes Attributes
	Value      map[string]interface{}
}

func ToAttributes(item ...string) Attributes {
	data := Attributes{}
	for i := 0; i < len(item)-1; i += 2 {
		data[item[i]] = item[i+1]
	}
	return data
}

func Filter(input map[string]Item, filter Attributes) map[string]Item {
	newMap := map[string]Item{}
	for k, item := range input {
		if compare(filter, item.Attributes) {
			newMap[k] = item
		}
	}
	return newMap
}
