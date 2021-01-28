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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilterOwner(t *testing.T) {
	testCases := []struct {
		Name                  string
		InputItems            map[string]OwnableItem
		InputOwner            string
		ExpectedFilteredItems map[string]OwnableItem
	}{
		{
			Name:                  "No items",
			InputOwner:            "Argus",
			ExpectedFilteredItems: make(map[string]OwnableItem),
		},
		{
			Name: "No owner - nothing filtered out",
			InputItems: map[string]OwnableItem{
				"item0": {},
				"item1": {},
			},
			ExpectedFilteredItems: map[string]OwnableItem{
				"item0": {},
				"item1": {},
			},
		},
		{
			Name:       "Filtered by owner",
			InputOwner: "Argus",
			InputItems: map[string]OwnableItem{
				"item0": {
					Owner: "Tr1d1um",
				},

				"item1": {
					Owner: "Argus",
				},
				"item2": {
					Owner: "Talaria",
				},
			},
			ExpectedFilteredItems: map[string]OwnableItem{
				"item1": {
					Owner: "Argus",
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			assert := assert.New(t)
			filteredItems := FilterOwner(testCase.InputItems, testCase.InputOwner)
			assert.Equal(testCase.ExpectedFilteredItems, filteredItems)
		})

	}
}
