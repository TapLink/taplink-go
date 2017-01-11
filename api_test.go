package taplink

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	testAppID    = "7ddf60de9250dce2f9f9a4ff1f5be257eb42e81d872a9381271edddae1fb83f2f99b89f138354fb8098d1e9b6681d6b0a58bbd2b26637b545c1c32607e85d7cf"
	errRespAppID = "First part of the path must be a 64-byte AppID, encoded as a 128-character hexidecimal string, e.g. '/<AppID>/'"
	errRespHash  = "Second part of the path must be a 64-byte Hash, encoded as a 128-character hexidecimal string, e.g. '/<AppID>/<Hash>/'"

	testHashString            = "7ddf60de9250dce2f9f9a4ff1f5be257eb42e81d872a9381271edddae1fb83f2f99b89f138354fb8098d1e9b6681d6b0a58bbd2b26637b545c1c32607e85d7cf"
	testHashBytes             = hexString(testHashString).Bytes()
	testHashExpectedSalt      = "edb8b9f2560a5bb7a354ca14c0dd72c377474fbad0afb9d73dd8fa01210777b995320979df40c7eab64450a7ef368ff8019350c613538f6abad9c4d9d8879bf5"
	testHashExpectedSaltBytes = hexString(testHashExpectedSalt).Bytes()

	testPasswordSumHashStr = "38a9799aaabfb4521417d4cc84a101523c2f933b7a583636591483aded3afc07b243ce96d49f6d0be86127cd738c80938676752669d323253c3f434c04191cad"

	origTransport = HTTPClient.Transport
)

type hexString string

func (s hexString) Bytes() []byte {
	b, _ := hex.DecodeString(string(s))
	return b
}

type testRoundTripper struct {
	code    int
	latency time.Duration
	headers map[string]string
	body    []byte
	err     error
}

func (rt *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.err != nil {
		return nil, rt.err
	}
	if rt.latency > time.Duration(0) {
		time.Sleep(rt.latency)
	}
	hdr := make(map[string][]string, 0)
	if rt.code > 200 && rt.body == nil {
		rt.body = []byte(http.StatusText(rt.code))
	}
	resp := &http.Response{
		StatusCode: rt.code,
		Status:     http.StatusText(rt.code),
		Body:       ioutil.NopCloser(bytes.NewBuffer(rt.body)),
		Header:     http.Header(hdr),
	}
	resp.Header.Set("X-TEST", "true")
	if rt.headers != nil {
		for k, v := range rt.headers {
			resp.Header.Set(k, v)
		}
	}
	return resp, nil
}

func TestNew(t *testing.T) {
	a := New(testAppID)
	assert.Equal(t, testAppID, a.Config().AppID())
	assert.Equal(t, "api.taplink.co", a.Config().Host(0))
}

func TestWithTestServer(t *testing.T) {
	HTTPClient.Transport = &testRoundTripper{503, 0, nil, nil, nil}
	defer func() {
		HTTPClient.Transport = origTransport
	}()
	c := New(testAppID).(*Client)
	_, err := c.getFromAPI("/foobar")
	assert.Equal(t, http.StatusText(503), err.Error())
}

func TestWithInvalidJSONResponse(t *testing.T) {
	HTTPClient.Transport = &testRoundTripper{200, 0, nil, []byte("foobar"), nil}
	defer func() {
		HTTPClient.Transport = origTransport
	}()
	c := New(testAppID).(*Client)
	_, err := c.getSalt([]byte(""), 0)
	assert.True(t, strings.HasPrefix(err.Error(), "invalid character"))
}

func TestWithInvalidHexStringResponse(t *testing.T) {
	HTTPClient.Transport = &testRoundTripper{200, 0, nil, []byte(`{"s2":"---invalid hex string here---","vid":3}`), nil}
	defer func() {
		HTTPClient.Transport = origTransport
	}()
	c := New(testAppID).(*Client)
	_, err := c.getSalt([]byte(""), 0)
	assert.Equal(t, hex.ErrLength, err)
}

func TestWithReadFailure(t *testing.T) {
	hdr := map[string]string{"Content-Length": "111111111"}
	HTTPClient.Transport = &testRoundTripper{200, 0, hdr, nil, nil}
	defer func() {
		HTTPClient.Transport = origTransport
	}()

	c := New(testAppID).(*Client)
	_, err := c.getFromAPI("/foo")
	assert.EqualError(t, err, "unexpected EOF")
}

func TestInvalidURL(t *testing.T) {
	c := New(testAppID).(*Client)
	_, err := c.getFromAPI("/foobar")
	assert.Error(t, err)
}

// TestHTTPClientFailure tests a request to a bogus server/port to ensure that
// the HTTPClient fails and the RetryLimit and RetryDelay are respected.
func TestHTTPClientFailure(t *testing.T) {
	HTTPClient.Transport = &testRoundTripper{503, 0, nil, nil, errors.New("test error")}
	defer func() {
		HTTPClient.Transport = origTransport
	}()
	c := New(testAppID).(*Client)
	c.Stats().Enable()
	// First attempt isn't delayed, so subtract 1 from the RetryLimit
	expectedTime := time.Now().Add(RetryDelay * time.Duration(RetryLimit-1))
	host := c.Config().Host(0)
	_, err := c.getFromAPI("/foobar")
	assert.NotNil(t, err)
	assert.Equal(t, int(RetryLimit), c.Stats().Get(host).Errors().Len())
	if !assert.True(t, time.Now().After(expectedTime)) {
		t.Logf("Expected now (%d) to be after %d", time.Now().Unix(), expectedTime.Unix())
	}
}

func TestInvalidRequest(t *testing.T) {
	c := New(testAppID).(*Client)
	_, err := c.getFromAPI("/foobar")
	assert.Error(t, err)
}

func TestIncErrs(t *testing.T) {
	c := New(testAppID).(*Client)
	host := c.Config().Host(0)
	c.Stats().Disable()
	c.Stats().AddError(host, 999)
	assert.Equal(t, 0, c.Stats().Get(host).Errors().Len())
	c.Stats().Enable()
	c.Stats().AddError(host, 999)
	assert.Equal(t, 1, c.Stats().Get(host).Errors().Len())
}

func TestIncErrsNoLatency(t *testing.T) {
	c := New(testAppID).(*Client)
	host := c.Config().Host(0)
	errCode := 503
	c.Stats().Enable()
	c.Stats().AddError(host, errCode)
	assert.Equal(t, 1, c.Stats().Get(host).Errors().Len())
	assert.Equal(t, 0, c.Stats().Get(host).Latency().Len())
}

func TestIncSuccess(t *testing.T) {
	c := New(testAppID).(*Client)
	host := c.Config().Host(0)
	c.Stats().Disable()
	c.Stats().AddSuccess(host, 10*time.Millisecond)
	assert.Equal(t, 0, c.Stats().Get(host).Latency().Len())
	c.Stats().Enable()
	c.Stats().AddSuccess(host, 10*time.Millisecond)
	assert.Equal(t, 1, c.Stats().Get(host).Latency().Len())
}

func TestGetSalt(t *testing.T) {
	c := New(testAppID).(*Client)
	c.Stats().Enable()
	host := c.Config().Host(0)
	s, err := c.getSalt(testHashBytes, 0)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, s.Salt, testHashExpectedSaltBytes)
	assert.Equal(t, int(1), c.Stats().Get(host).Requests())
	assert.Equal(t, testHashExpectedSalt, fmt.Sprintf("%s", s))
}

func TestGetSaltErr(t *testing.T) {
	c := New(testAppID).(*Client)
	s, err := c.getSalt(nil, 0)
	assert.Nil(t, s)
	assert.Error(t, err)
	assert.EqualError(t, err, errRespHash)
}

func TestNewPassword(t *testing.T) {
	c := New(testAppID).(*Client)
	p, err := c.NewPassword(testHashBytes)
	assert.NoError(t, err)

	// Get a hash of the expected salt and the input password
	sum := hmac.New(sha512.New, testHashExpectedSaltBytes)
	sum.Write(testHashBytes)
	assert.Equal(t, p.Hash, sum.Sum(nil))
	assert.Equal(t, testPasswordSumHashStr, fmt.Sprintf("%s", p))
}

func TestNewPasswordInvalid(t *testing.T) {
	c := New(testAppID).(*Client)
	p, err := c.NewPassword(nil)
	assert.Error(t, err)
	assert.Nil(t, p)
	assert.EqualError(t, err, errRespHash)
}

func TestVerifyPassword(t *testing.T) {
	c := New(testAppID).(*Client)
	p, err := c.NewPassword(testHashBytes)
	if !assert.NoError(t, err) {
		return
	}

	v, err := c.VerifyPassword(testHashBytes, p.Hash, 0)
	assert.NoError(t, err)
	assert.NotNil(t, v)
	assert.True(t, v.Matched)
	assert.Equal(t, testPasswordSumHashStr, fmt.Sprintf("%s", v))
}

func TestVerifyPasswordNewVersion(t *testing.T) {
	c := New(testAppID).(*Client)

	// Get the old expected. Need to use the older version of getSalt for that.
	// Cannot depend on NewPassword because it uses the latest version.
	salt, err := c.getSalt(testHashBytes, 2)
	if !assert.NoError(t, err) {
		return
	}

	prevSum := hmac.New(sha512.New, salt.Salt)
	prevSum.Write(testHashBytes)
	prevExpected := prevSum.Sum(nil)

	v, err := c.VerifyPassword(testHashBytes, prevExpected, 2)
	if !assert.NoError(t, err) {
		return
	}

	// Now get the expected values for the new version. These will then be comparted to
	// the VerifyPassword NewHash field.
	p, err := c.NewPassword(testHashBytes)
	if !assert.NoError(t, err) {
		return
	}

	assert.NoError(t, err)
	assert.NotNil(t, v)
	assert.True(t, v.Matched)
	assert.Equal(t, p.Hash, v.NewHash)
	assert.Equal(t, int64(2), v.VersionID)
	assert.Equal(t, int64(3), v.NewVersionID)
}

func TestVerifyPasswordError(t *testing.T) {
	c := New(testAppID).(*Client)
	p, err := c.VerifyPassword([]byte("foobar"), nil, 0)
	assert.Error(t, err)
	assert.Nil(t, p)
}

func TestVerifyPasswordFail(t *testing.T) {
	c := New(testAppID).(*Client)
	p, err := c.VerifyPassword(testHashBytes, []byte("foobar"), 0)
	assert.NoError(t, err)
	assert.NotNil(t, p)
	assert.False(t, p.Matched)
}

func TestVersionID(t *testing.T) {
	assert.Equal(t, "", fmt.Sprintf("%s", Version(0)))
	assert.Equal(t, "1", fmt.Sprintf("%s", Version(1)))
}

// TestVectorsV3 runs tests for correctness of the results vs. known values
func TestVectorsV3(t *testing.T) {

	sum := hmac.New(sha512.New, hexString("4cb78a1a60599df9c3bd9e4ac741a5f15feec1812b22a5f15bbad978039f2765f00dd82d97272eb3674cd164a0cc7024bbfd3704c6df6e2cb17a6562bd96ecb7").Bytes())
	sum.Write([]byte("secret"))
	hash1 := sum.Sum(nil)

	c := New(testAppID).(*Client)
	p, err := c.NewPassword(hash1)
	assert.NoError(t, err)
	assert.Equal(t, hexString("9a4893d65a8eec23e520d0c7abe9c170ba61548c754b4805226e48d7519c55ed7f0daec920c5a99019042745007b99822e6853b8620be67955610b6d25f4b2f9").Bytes(), p.Hash)

	s, err := c.getSalt(hash1, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), s.VersionID)
	assert.Equal(t, hexString("080b64a980fe49664e6e29e7532ce4dab19a070da0618e32b20d7d0578e120458c1fcf7f3de0a9da7bbf7ba49cacabc05230c605f7032ab51323992ff3c35895").Bytes(), s.Salt)
	assert.Equal(t, int64(0), s.NewVersionID)
	assert.Nil(t, s.NewSalt)

	sum = hmac.New(sha512.New, s.Salt)
	sum.Write(hash1)
	assert.Equal(t, hexString("9a4893d65a8eec23e520d0c7abe9c170ba61548c754b4805226e48d7519c55ed7f0daec920c5a99019042745007b99822e6853b8620be67955610b6d25f4b2f9").Bytes(), sum.Sum(nil))
}

func TestVectorsV2(t *testing.T) {

	c := New(testAppID).(*Client)

	sum := hmac.New(sha512.New, hexString("4cb78a1a60599df9c3bd9e4ac741a5f15feec1812b22a5f15bbad978039f2765f00dd82d97272eb3674cd164a0cc7024bbfd3704c6df6e2cb17a6562bd96ecb7").Bytes())
	sum.Write([]byte("secret"))
	hash1 := sum.Sum(nil)

	s, err := c.getSalt(hash1, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), s.VersionID)
	assert.Equal(t, hexString("6190928f03b4ca59aed71614876857679e1edcf9b03ce3443a006713bcb2a305d33ee250c327df00f946041ca435a2cf72dd421e02f1e0d8de3efd5406674f6f").Bytes(), s.Salt)
	assert.Equal(t, int64(3), s.NewVersionID)
	assert.Equal(t, hexString("080b64a980fe49664e6e29e7532ce4dab19a070da0618e32b20d7d0578e120458c1fcf7f3de0a9da7bbf7ba49cacabc05230c605f7032ab51323992ff3c35895").Bytes(), s.NewSalt)

	sum = hmac.New(sha512.New, s.Salt)
	sum.Write(hash1)
	hash2 := sum.Sum(nil)
	assert.Equal(t, hexString("d883c376526904dd90bd69709d259e7d4ac4fe1ee3ff65a2b6ed2920c8baad326b0c2043c6bb7750c6ad02284c2365d3c61298649107924cc44e60450031fbd2").Bytes(), hash2)

	p, err := c.VerifyPassword(hash1, hash2, 2)
	if !assert.NoError(t, err) {
		return
	}
	assert.True(t, p.Matched)
	assert.Equal(t, int64(3), p.NewVersionID)
	assert.Equal(t, hexString("9a4893d65a8eec23e520d0c7abe9c170ba61548c754b4805226e48d7519c55ed7f0daec920c5a99019042745007b99822e6853b8620be67955610b6d25f4b2f9").Bytes(), p.NewHash)
}

// BenchmarkGetSalt tests parallel performance of getting multiple salts from a single client
// To avoid making requests over the network, a pre-defined response is set.
func BenchmarkGetSalt(b *testing.B) {

	HTTPClient.Transport = &testRoundTripper{200, 0, nil, []byte(`{"s2":"edb8b9f2560a5bb7a354ca14c0dd72c377474fbad0afb9d73dd8fa01210777b995320979df40c7eab64450a7ef368ff8019350c613538f6abad9c4d9d8879bf5","vid":3}`), nil}
	defer func() {
		HTTPClient.Transport = origTransport
	}()

	var i int
	var mu sync.Mutex
	c := New(testAppID).(*Client)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.getSalt(testHashBytes, 0)
			mu.Lock()
			i++
			mu.Unlock()
		}
	})
	b.Logf("Processed %d requests", i)
}

// BenchmarkGetSaltNetwork runs
func BenchmarkGetSaltNetwork(b *testing.B) {
	var i int
	var mu sync.Mutex
	c := New(testAppID).(*Client)
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s, err := c.getSalt(testHashBytes, 0)
			if err != nil {
				b.Fail()
			}
			if !bytes.Equal(testHashExpectedSaltBytes, s.Salt) {
				b.Fail()
			}
			mu.Lock()
			i++
			mu.Unlock()
		}
	})
	b.Logf("Sent %d requests", i)
}
