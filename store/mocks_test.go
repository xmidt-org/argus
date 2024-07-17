// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package store

import (
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/ancla/model"
)

type MockDAO struct {
	mock.Mock
}

func (m *MockDAO) Push(key model.Key, item OwnableItem) error {
	args := m.Called(key, item)
	return args.Error(0)
}

func (m *MockDAO) Get(key model.Key) (OwnableItem, error) {
	args := m.Called(key)
	return args.Get(0).(OwnableItem), args.Error(1)
}

func (m *MockDAO) Delete(key model.Key) (OwnableItem, error) {
	args := m.Called(key)
	return args.Get(0).(OwnableItem), args.Error(1)
}

func (m *MockDAO) GetAll(bucket string) (map[string]OwnableItem, error) {
	args := m.Called(bucket)
	return args.Get(0).(map[string]OwnableItem), args.Error(1)
}
