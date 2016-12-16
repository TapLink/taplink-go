package taplink

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	testAppID    = "7ddf60de9250dce2f9f9a4ff1f5be257eb42e81d872a9381271edddae1fb83f2f99b89f138354fb8098d1e9b6681d6b0a58bbd2b26637b545c1c32607e85d7cf"
	errRespAppID = "First part of the path must be a 64-byte AppID, encoded as a 128-character hexidecimal string, e.g. '/<AppID>/'"
	errRespHash  = "Second part of the path must be a 64-byte Hash, encoded as a 128-character hexidecimal string, e.g. '/<AppID>/<Hash>/'"
	mockCfgResp  = `{"lastModified":1481831132236,"servers":["api.taplink.co","api-us-west.taplink.co"]}`

	testHashString            = "7ddf60de9250dce2f9f9a4ff1f5be257eb42e81d872a9381271edddae1fb83f2f99b89f138354fb8098d1e9b6681d6b0a58bbd2b26637b545c1c32607e85d7cf"
	testHashBytes             []byte
	testHashExpectedSalt      = "edb8b9f2560a5bb7a354ca14c0dd72c377474fbad0afb9d73dd8fa01210777b995320979df40c7eab64450a7ef368ff8019350c613538f6abad9c4d9d8879bf5"
	testHashExpectedSaltBytes []byte

	testPasswordSumHashStr = "38a9799aaabfb4521417d4cc84a101523c2f933b7a583636591483aded3afc07b243ce96d49f6d0be86127cd738c80938676752669d323253c3f434c04191cad"
)

func init() {
	testHashBytes, _ = hex.DecodeString(testHashString)
	testHashExpectedSaltBytes, _ = hex.DecodeString(testHashExpectedSalt)
}

func TestNew(t *testing.T) {
	a := New(testAppID)
	assert.Equal(t, testAppID, a.Config().AppID())
	assert.Equal(t, "https://api.taplink.co", a.Config().Host())
}

func TestIncErrs(t *testing.T) {
	c := New(testAppID).(*Client)
	assert.False(t, c.stats)
	c.incrErrs(10 * time.Millisecond)
	assert.Equal(t, int64(0), c.reqErrCt)
	assert.Len(t, c.reqLatency, 0)
	c.EnableStats()
	c.incrErrs(10 * time.Millisecond)
	assert.Equal(t, int64(1), c.reqErrCt)
	assert.Len(t, c.reqLatency, 1)
}

func TestIncErrsNoLatency(t *testing.T) {
	c := New(testAppID).(*Client)
	c.EnableStats()
	c.incrErrs(0)
	assert.Equal(t, int64(1), c.reqErrCt)
	assert.Len(t, c.reqLatency, 0)
}

func TestIncSuccess(t *testing.T) {
	c := New(testAppID).(*Client)
	assert.False(t, c.stats)
	c.incrSuccess(10 * time.Millisecond)
	assert.Equal(t, int64(0), c.reqCt)
	assert.Len(t, c.reqLatency, 0)
	c.EnableStats()
	c.incrSuccess(10 * time.Millisecond)
	assert.Equal(t, int64(1), c.reqCt)
	assert.Len(t, c.reqLatency, 1)
}

func TestGetSalt(t *testing.T) {
	c := New(testAppID).(*Client)
	c.EnableStats()
	s, err := c.getSalt(testHashBytes, 0)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, s.Salt, testHashExpectedSaltBytes)
	assert.Equal(t, int64(1), c.Requests())
	assert.Equal(t, testHashExpectedSalt, fmt.Sprintf("%s", s))
}

func TestGetSaltErr(t *testing.T) {
	c := New(testAppID).(*Client)
	c.EnableStats()
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
