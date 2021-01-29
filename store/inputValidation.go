package store

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"

	"github.com/xmidt-org/argus/model"
)

var (
	idFormatRegex     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	bucketFormatRegex = regexp.MustCompile("^[0-9a-z][0-9a-z-]{1,61}[0-9a-z]$")
)

var (
	errInvalidID               = BadRequestErr{Message: "Invalid ID format. Expecting the format of a SHA-256 message digest."}
	errIDMismatch              = BadRequestErr{Message: "IDs must match between the URL and payload."}
	errDataFieldMissing        = BadRequestErr{Message: "Data field must be set in payload."}
	errBodyReadFailure         = BadRequestErr{Message: "Failed to read body."}
	errPayloadUnmarshalFailure = BadRequestErr{Message: "Failed to unmarshal json payload."}
	errInvalidBucket           = BadRequestErr{Message: "Invalid bucket format."}
)

func validateItemTTL(item *model.Item, maxTTL time.Duration) {
	ttlMaxSeconds := int64(maxTTL.Seconds())
	if item.TTL == nil || *item.TTL > ttlMaxSeconds {
		item.TTL = &ttlMaxSeconds
	}
}

// isIDValid returns true if the given ID is a hex digest string of 64 characters (i.e. 7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7).
// False otherwise. Note that per the input string name, we expect the ID to be normalized by the time it gets here (remove whitespaces, all lowercase)
func isIDValid(normalizedID string) bool {
	return idFormatRegex.MatchString(normalizedID)
}

// isBucketValid return true if and only if all the following rules are satisfied. False otherwise.
// 1) Between 3 and 63 characters long.
// 2) Consists only of lowercase letters, numbers and hyphens (-).
// 3) Must begin and end with a letter or number.
func isBucketValid(bucket string) bool {
	return bucketFormatRegex.MatchString(bucket)
}

// validateItemPathVars returns a pertinent HTTP-coded error if any of the input variables
// are invalid, nil otherwise.
func validateItemPathVars(bucket, normalizedID string) error {
	if !isIDValid(normalizedID) {
		return errInvalidID
	}

	if !isBucketValid(bucket) {
		return errInvalidBucket
	}

	return nil
}

// validItemUnmarshaler ensures that the unmarshaled item based on
// the URL ID and configuration constraints.
type validItemUnmarshaler struct {
	item   model.Item
	id     string
	config *transportConfig
}

func (v *validItemUnmarshaler) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &v.item); err != nil {
		return err
	}

	if len(v.item.Data) < 1 {
		return errDataFieldMissing
	}

	if !isIDValid(v.item.ID) {
		return errInvalidID
	}

	v.item.ID = strings.ToLower(v.item.ID)

	if v.item.ID != v.id {
		return errIDMismatch
	}

	validateItemTTL(&v.item, v.config.ItemMaxTTL)

	return nil
}
