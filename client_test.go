package taplink

import (
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testNetTOErr string

func (e testNetTOErr) Error() string {
	return string(e)
}

func (e testNetTOErr) Timeout() bool {
	return true
}

func (e testNetTOErr) Temporary() bool {
	return true
}

func TestGetFromClientTimeoutError(t *testing.T) {
	HTTPClient.Transport = &testRoundTripper{200, 0, nil, []byte("foobar"), testNetTOErr("test timeout")}
	defer func() {
		HTTPClient.Transport = origTransport
	}()
	c := New(testAppID).(*Client)
	c.Stats().Enable()

	_, err := c.getFromAPI("/foobar")
	assert.Error(t, err)
	ne, ok := err.(net.Error)
	if assert.True(t, ok) {
		return
	}
	assert.True(t, ne.Timeout())
	assert.True(t, ne.Temporary())
	assert.Equal(t, int(RetryLimit), c.Stats().Get(DefaultHost).Timeouts())
}

func TestGetFromClientServerErr(t *testing.T) {
	HTTPClient.Transport = &testRoundTripper{500, 0, nil, []byte(http.StatusText(http.StatusInternalServerError)), nil}
	defer func() {
		HTTPClient.Transport = origTransport
	}()
	c := New(testAppID).(*Client)
	c.Stats().Enable()

	_, err := c.getFromAPI("/foobar")
	t.Logf("Error: %v\n", err)
	assert.Error(t, err)
	assert.Equal(t, int(RetryLimit), c.Stats().Get(DefaultHost).Errors().Count(500))
	assert.Equal(t, int(RetryLimit), c.Stats().Get(DefaultHost).Errors().Len())
}

func TestGetFromClientClientErr(t *testing.T) {
	code := http.StatusUnauthorized
	HTTPClient.Transport = &testRoundTripper{code, 0, nil, []byte(http.StatusText(code)), nil}
	defer func() {
		HTTPClient.Transport = origTransport
	}()
	c := New(testAppID).(*Client)
	c.Stats().Enable()

	_, err := c.getFromAPI("/foobar")
	assert.EqualError(t, err, http.StatusText(code))
	assert.Equal(t, int(1), c.Stats().Get(DefaultHost).Errors().Count(code))
	assert.Equal(t, int(1), c.Stats().Get(DefaultHost).Errors().Len())
}
