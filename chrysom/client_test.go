package chrysom

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInterface(t *testing.T) {
	assert := assert.New(t)
	var (
		client interface{}
	)
	client = &Client{}
	_, ok := client.(Pusher)
	assert.True(ok, "not a pusher")
	_, ok = client.(Reader)
	assert.True(ok, "not a reader")
}
