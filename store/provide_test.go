// SPDX-FileCopyrightText: 2021 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0
package store

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTransportConfig(t *testing.T) {
	type testCase struct {
		Description             string
		UserInputValConfig      UserInputValidationConfig
		ExpectedTransportConfig transportConfig
		ShouldUnmarshalFail     bool
		ExpectedErr             error
	}

	tcs := []testCase{
		{
			Description: "Bad regex for bucket",
			ExpectedErr: errRegexCompilation,
			UserInputValConfig: UserInputValidationConfig{
				BucketFormatRegex: "??",
				OwnerFormatRegex:  ".*",
			},
		},
		{
			Description: "Bad regex for owner",
			ExpectedErr: errRegexCompilation,
			UserInputValConfig: UserInputValidationConfig{
				OwnerFormatRegex:  "??",
				BucketFormatRegex: ".*",
			},
		},
		{
			Description:             "Default values",
			UserInputValConfig:      UserInputValidationConfig{},
			ExpectedTransportConfig: getDefaultValuesExpectedConfig(),
		},
		{
			Description: "Check values",
			UserInputValConfig: UserInputValidationConfig{
				ItemMaxTTL:        48 * time.Hour,
				BucketFormatRegex: ".+",
				OwnerFormatRegex:  ".*",
				ItemDataMaxDepth:  5,
			},
			ExpectedTransportConfig: getCheckValuesExpectedConfig(),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			transportConfig, err := newTransportConfig(transportConfigIn{
				AccessLevelAttributeKey: "attr-key",
				UserInputValidation:     tc.UserInputValConfig,
			})
			if tc.ExpectedErr == nil {
				require.Nil(err)
				require.NotNil(transportConfig)
				assert.Equal(tc.ExpectedTransportConfig, *transportConfig)
			} else {
				errors.Is(err, tc.ExpectedErr)
			}
		})
	}
}

func getDefaultValuesExpectedConfig() transportConfig {
	return transportConfig{
		AccessLevelAttributeKey: "attr-key",
		ItemMaxTTL:              time.Hour * 24,
		OwnerFormatRegex:        regexp.MustCompile(OwnerFormatRegexSource),
		IDFormatRegex:           regexp.MustCompile(IDFormatRegexSource),
		BucketFormatRegex:       regexp.MustCompile(BucketFormatRegexSource),
		ItemDataMaxDepth:        defaultItemDataMaxDepth,
	}
}

func getCheckValuesExpectedConfig() transportConfig {
	return transportConfig{
		AccessLevelAttributeKey: "attr-key",
		ItemMaxTTL:              time.Hour * 48,
		OwnerFormatRegex:        regexp.MustCompile(".*"),
		IDFormatRegex:           regexp.MustCompile(IDFormatRegexSource),
		BucketFormatRegex:       regexp.MustCompile(".+"),
		ItemDataMaxDepth:        5,
	}
}
