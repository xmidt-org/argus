// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package store

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/xmidt-org/argus/auth"
	"go.uber.org/fx"
)

var errRegexCompilation = errors.New("regex could not be compiled")

// allow up to 31 nested objects in item data by default
const defaultItemDataMaxDepth uint = 30

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
	AccessLevel auth.AccessLevel
}

func newAccessLevelAttributeKeyAnnotated() fx.Annotated {
	return fx.Annotated{
		Name: "access_level_attribute_key",
		Target: func(in accessLevelAttributeKeyIn) string {
			return in.AccessLevel.AttributeKey
		},
	}
}

type UserInputValidationConfig struct {
	ItemMaxTTL        time.Duration
	BucketFormatRegex string
	OwnerFormatRegex  string
	ItemDataMaxDepth  uint
}

type transportConfigIn struct {
	fx.In
	UserInputValidation     UserInputValidationConfig
	AccessLevelAttributeKey string `name:"access_level_attribute_key"`
}

func newTransportConfig(in transportConfigIn) (*transportConfig, error) {
	v := in.UserInputValidation

	if v.ItemMaxTTL == 0 {
		v.ItemMaxTTL = time.Hour * 24
	}

	if v.ItemDataMaxDepth == 0 {
		v.ItemDataMaxDepth = defaultItemDataMaxDepth
	}

	config := &transportConfig{
		AccessLevelAttributeKey: in.AccessLevelAttributeKey,
		ItemMaxTTL:              v.ItemMaxTTL,
		ItemDataMaxDepth:        v.ItemDataMaxDepth,
	}

	err := buildInputRegexValidators(v, config)
	return config, err
}

// useOrDefault returns the value if it's not the empty string. Otherwise, it returns the defaultValue.
func useOrDefault(value, defaultValue string) string {
	if len(value) > 0 {
		return value
	}
	return defaultValue
}

func buildInputRegexValidators(userInputValidation UserInputValidationConfig, config *transportConfig) error {
	bucketFormatRegex := useOrDefault(userInputValidation.BucketFormatRegex, BucketFormatRegexSource)
	bucketRegex, err := regexp.Compile(bucketFormatRegex)
	if err != nil {
		return fmt.Errorf("bucket %w: %v", errRegexCompilation, err)
	}
	config.BucketFormatRegex = bucketRegex

	ownerFormatRegex := useOrDefault(userInputValidation.OwnerFormatRegex, OwnerFormatRegexSource)
	ownerRegex, err := regexp.Compile(ownerFormatRegex)
	if err != nil {
		return fmt.Errorf("owner %w: %v", errRegexCompilation, err)
	}
	config.OwnerFormatRegex = ownerRegex

	config.IDFormatRegex = regexp.MustCompile(IDFormatRegexSource)
	return nil
}
