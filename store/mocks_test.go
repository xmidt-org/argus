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
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/model"
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
