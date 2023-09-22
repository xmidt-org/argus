// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
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
