package store

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsIDValid(t *testing.T) {
	type test struct {
		Name     string
		ID       string
		Expected bool
	}
	IDFormatRegex := regexp.MustCompile(IDFormatRegexSource)

	tcs := []test{
		{Name: "CharacterOver", ID: "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7a", Expected: false},
		{Name: "NonHex", ID: "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7z", Expected: false},
		{Name: "NonLowerCase", ID: "7E8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7", Expected: false},
		{Name: "Success", ID: "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7", Expected: true},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(tc.Expected, isIDValid(IDFormatRegex, tc.ID))
		})
	}
}

func TestIsBucketValidFromDefaultSource(t *testing.T) {
	type testCase struct {
		Description string
		Bucket      string
		Succeeds    bool
	}

	BucketFormatRegex := regexp.MustCompile(BucketFormatRegexSource)

	tcs := []testCase{
		{
			Description: "Too short",
			Bucket:      "ab",
		},
		{
			Description: "Too long",
			Bucket:      "neque-porro-quisquam-est-qui-dolorem-ipsum-quia-dolor-sit-amet-c",
		},
		{
			Description: "Bad start",
			Bucket:      "?this-could-ve-been-fine-but",
		},
		{
			Description: "Bad end",
			Bucket:      "this-could-ve-also-been-fine-but-",
		},
		{
			Description: "Success",
			Bucket:      "a-nice-readable-bucket-indeed",
			Succeeds:    true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(tc.Succeeds, isBucketValid(BucketFormatRegex, tc.Bucket))
		})
	}
}

func TestIsOwnerValidFromDefaultSource(t *testing.T) {
	type testCase struct {
		Description string
		Owner       string
		Succeeds    bool
	}

	OwnerFormatRegex := regexp.MustCompile(OwnerFormatRegexSource)

	tcs := []testCase{
		{
			Description: "Too short",
			Owner:       "xmidt",
		},
		{
			Description: "Too long",
			Owner:       "neque-porro-quisquam-est-qui-dolorem-ipsum-quia-dolor-sit-amet-c",
		},
		{
			Description: "Owner is optional",
			Owner:       "",
			Succeeds:    true,
		},
		{
			Description: "Success",
			Owner:       "a-nice-readable-owner-indeed",
			Succeeds:    true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(tc.Succeeds, isOwnerValid(OwnerFormatRegex, tc.Owner))
		})
	}
}

func TestValidDepth(t *testing.T) {
	tcs := []struct {
		Description   string
		InputJSONData string
		MaxDepth      uint
		IsValid       bool
	}{
		{
			Description:   "Well within constraints",
			InputJSONData: `{"hey": "jude"}`,
			MaxDepth:      3,
			IsValid:       true,
		},

		{
			Description:   "Min depth edge case",
			InputJSONData: `{"hey": "jude"}`,
			MaxDepth:      1,
			IsValid:       true,
		},

		{
			Description:   "At depth limit",
			InputJSONData: `{"hey": {"jude": "don't be afraid"}}`,
			MaxDepth:      2,
			IsValid:       true,
		},
		{
			Description:   "Pass depth limits",
			InputJSONData: `{"hey": {"jude": {"don't make it bad": {"take a sad song & make it better": 3}}}}`,
			MaxDepth:      3,
			IsValid:       false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Description, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)
			var data map[string]interface{}
			err := json.Unmarshal([]byte(tc.InputJSONData), &data)
			require.Nil(err)

			isValid := validDepth(data, tc.MaxDepth)
			assert.Equal(tc.IsValid, isValid)
		})
	}
}
