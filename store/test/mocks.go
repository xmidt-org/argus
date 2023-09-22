// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/argus/store"
)

type MockDB struct {
	mock.Mock
}

func (s *MockDB) Push(key model.Key, item store.OwnableItem) error {
	args := s.Called(key, item)
	return args.Error(0)
}

func (s *MockDB) Get(key model.Key) (store.OwnableItem, error) {
	args := s.Called(key)
	return args.Get(0).(store.OwnableItem), args.Error(1)
}

func (s *MockDB) Delete(key model.Key) (store.OwnableItem, error) {
	args := s.Called(key)
	return args.Get(0).(store.OwnableItem), args.Error(1)
}

func (s *MockDB) GetAll(bucket string) (map[string]store.OwnableItem, error) {
	args := s.Called(bucket)
	return args.Get(0).(map[string]store.OwnableItem), args.Error(1)
}

func (s *MockDB) Close() {
	s.Called()
}

func (s *MockDB) Ping() error {
	args := s.Called()
	return args.Error(0)
}
