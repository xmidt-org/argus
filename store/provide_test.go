package store

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/themis/config"
)

var errTestUnmarshalFail = errors.New("I was told to fail on UnmarshalKey")

func TestNewTransportConfig(t *testing.T) {
	type testCase struct {
		Description             string
		UserInputValConfig      userInputValidationConfig
		ExpectedTransportConfig transportConfig
		ShouldUnmarshalFail     bool
		ExpectedErr             error
	}

	var itemDataMaxDepth uint = 5

	tcs := []testCase{
		{
			Description:         "Unmarshal fails",
			ShouldUnmarshalFail: true,
			ExpectedErr:         errTestUnmarshalFail,
		},
		{
			Description: "Bad regex for bucket",
			ExpectedErr: errRegexCompilation,
			UserInputValConfig: userInputValidationConfig{
				BucketFormatRegex: "??",
				OwnerFormatRegex:  ".*",
			},
		},
		{
			Description: "Bad regex for owner",
			ExpectedErr: errRegexCompilation,
			UserInputValConfig: userInputValidationConfig{
				OwnerFormatRegex:  "??",
				BucketFormatRegex: ".*",
			},
		},
		{
			Description:             "Default values",
			UserInputValConfig:      userInputValidationConfig{},
			ExpectedTransportConfig: getDefaultValuesExpectedConfig(),
		},
		{
			Description: "Check values",
			UserInputValConfig: userInputValidationConfig{
				ItemMaxTTL:        48 * time.Hour,
				BucketFormatRegex: ".+",
				OwnerFormatRegex:  ".*",
				ItemDataMaxDepth:  &itemDataMaxDepth,
			},
			ExpectedTransportConfig: getCheckValuesExpectedConfig(),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)
			tu := testUnmarshaler{
				userInputValConfig:  tc.UserInputValConfig,
				assert:              assert,
				require:             require,
				shouldUnmarshalFail: tc.ShouldUnmarshalFail,
			}

			transportConfig, err := newTransportConfig(transportConfigIn{
				AccessLevelAttributeKey: "attr-key",
				Unmarshaler:             tu,
			})
			if tc.ExpectedErr == nil {
				require.Nil(err)
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

type testUnmarshaler struct {
	config.Unmarshaller
	assert              *assert.Assertions
	require             *require.Assertions
	userInputValConfig  userInputValidationConfig
	shouldUnmarshalFail bool
}

func (t testUnmarshaler) UnmarshalKey(key string, value interface{}) error {
	t.assert.Equal("userInputValidation", key)
	userInputValidationConfig, ok := value.(*userInputValidationConfig)
	t.require.True(ok)

	if t.shouldUnmarshalFail {
		return errTestUnmarshalFail
	}
	*userInputValidationConfig = t.userInputValConfig
	return nil
}
