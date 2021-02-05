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
	"regexp"
	"time"

	"github.com/xmidt-org/argus/auth"
	"github.com/xmidt-org/themis/config"
	"go.uber.org/fx"
)

type handlerIn struct {
	fx.In

	Store  S
	Config *transportConfig
}

// ProvideHandlers fetches all dependencies and builds the four main handlers for this store.
func ProvideHandlers() fx.Option {
	return fx.Provide(
		newAccessLevelAttributeKeyAnnotated(),
		newTransportConfig,

		fx.Annotated{
			Name:   "set_handler",
			Target: newSetItemHandler,
		},
		fx.Annotated{
			Name:   "get_handler",
			Target: newGetItemHandler,
		},
		fx.Annotated{
			Name:   "get_all_handler",
			Target: newGetAllItemsHandler,
		},
		fx.Annotated{
			Name:   "delete_handler",
			Target: newDeleteItemHandler,
		},
	)
}

type accessLevelAttributeKeyIn struct {
	fx.In
	AccessLevel auth.AccessLevel `name:"primary_bearer_access_level"`
}

func newAccessLevelAttributeKeyAnnotated() fx.Annotated {
	return fx.Annotated{
		Name: "access_level_attribute_key",
		Target: func(in accessLevelAttributeKeyIn) string {
			return in.AccessLevel.AttributeKey
		},
	}
}

type userInputValidationConfig struct {
	ItemMaxTTL        time.Duration
	BucketFormatRegex string
	OwnerFormatRegex  string
}

type transportConfigIn struct {
	fx.In
	Unmarshaler             config.Unmarshaller
	AccessLevelAttributeKey string `name:"access_level_attribute_key"`
}

func newTransportConfig(in transportConfigIn) (*transportConfig, error) {
	var userInputValidation userInputValidationConfig

	if err := in.Unmarshaler.UnmarshalKey("userInputValidation", &userInputValidation); err != nil {
		return nil, err
	}

	if userInputValidation.ItemMaxTTL == 0 {
		userInputValidation.ItemMaxTTL = time.Hour * 24
	}

	config := &transportConfig{
		AccessLevelAttributeKey: in.AccessLevelAttributeKey,
		ItemMaxTTL:              userInputValidation.ItemMaxTTL,
	}

	buildInputRegexValidators(userInputValidation, config)
	return config, nil
}

func buildInputRegexValidators(userInputValidation userInputValidationConfig, config *transportConfig) error {
	if userInputValidation.BucketFormatRegex != "" {
		bucketRegex, err := regexp.Compile(userInputValidation.BucketFormatRegex)
		if err != nil {
			return err
		}
		config.BucketFormatRegex = bucketRegex
	}

	if userInputValidation.OwnerFormatRegex != "" {
		ownerRegex, err := regexp.Compile(userInputValidation.OwnerFormatRegex)
		if err != nil {
			return err
		}
		config.OwnerFormatRegex = ownerRegex
	}

	idFormatRegex, err := regexp.Compile(IDFormatRegexSource)

	if err != nil {
		return err
	}

	config.IDFormatRegex = idFormatRegex
	return nil
}
