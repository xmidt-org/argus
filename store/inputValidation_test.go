package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeID(t *testing.T) {
	type test struct {
		Name     string
		ID       string
		Expected string
	}

	tcs := []test{
		{Name: "Same", ID: "notchanged", Expected: "notchanged"},
		{Name: "ClearWhiteSpace", ID: "			clean    ", Expected: "clean"},
		{Name: "Lower case", ID: "TESTING!!	", Expected: "testing!!"},
		{Name: "Combined", ID: "			hElLo, WoRlD!    ", Expected: "hello, world!"},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(tc.Expected, normalizeID(tc.ID))
		})
	}
}

func TestIsIDValid(t *testing.T) {
	type test struct {
		Name     string
		ID       string
		Expected bool
	}

	tcs := []test{
		{Name: "CharacterOver", ID: "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7a", Expected: false},
		{Name: "NonHex", ID: "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7z", Expected: false},
		{Name: "NonLowerCase", ID: "7E8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7", Expected: false},
		{Name: "Success", ID: "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7", Expected: true},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			assert := assert.New(t)
			assert.Equal(tc.Expected, isIDValid(tc.ID))
		})
	}
}

func TestIsBucketValid(t *testing.T) {
	type testCase struct {
		Description string
		Bucket      string
		ExpectedErr error
		Succeeds    bool
	}

	tcs := []testCase{
		{
			Description: "Too short",
			Bucket:      "ab",
			ExpectedErr: errInvalidBucket,
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
			assert.Equal(tc.Succeeds, isBucketValid(tc.Bucket))
		})
	}
}
