package dtrack

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("return an error when creating a client with invalid URL", func(t *testing.T) {
		client, err := NewClient("https://invalid url.com", "token")

		var e url.InvalidHostError
		assert.ErrorAs(t, err, &e)
		assert.Nil(t, client)
	})

	t.Run("return an error when creating a client with empty API token", func(t *testing.T) {
		client, err := NewClient("https://url.com", "")

		assert.NotNil(t, err)
		assert.Nil(t, client)
	})

	t.Run("apply request timeout option when it's set", func(t *testing.T) {
		const timeout = 666
		client, _ := NewClient("https://url.com", "token", WithRequestTimeout(timeout))

		assert.Equal(t, time.Second*time.Duration(timeout), client.requestTimeout)
	})
}
