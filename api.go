package taplink

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	// ensures the Client implements the API interface
	_ API = (*Client)(nil)

	// DefaultTimeout is the default HTTP request timeout
	DefaultTimeout = 30 * time.Second
	// DefaultKeepAlive is the default HTTP keep-alive duration
	DefaultKeepAlive = 30 * time.Second

	// RetryLimit indicates how many times a connection should be retried before failing
	RetryLimit = 3
	// RetryDelay is the duration to wait between retry attempts
	RetryDelay = 1 * time.Second
)

// API is an interface which exposes TapLink API functionality
type API interface {

	// Config
	Config() Configuration

	// API funcs
	VerifyPassword(hash []byte, expectedHash []byte, versionID int64) (*VerifyPassword, error)
	NewPassword(hash []byte) (*NewPassword, error)

	// Requests returns the total number of HTTP requests made to the TapLink API, including those with errors and those without
	Requests() int64

	// Errors returns the total number of HTTP requests made to the TapLink API which ended in error
	Errors() int64

	// Latency returns the average latency of requests made to the TapLink API
	Latency() time.Duration

	// ErrorPct returns the pct of requests made to the TapLink API which ended in error.
	ErrorPct() int64

	// EnableStats starts the collection of stats regarding HTTP requests made to the TapLink API
	EnableStats()

	// DisableStats starts the collection of stats regarding HTTP requests made to the TapLink API
	DisableStats()
}

// Client is a struct which implements the API interface
type Client struct {
	cfg             Configuration
	reqCt, reqErrCt int64
	reqLatency      []time.Duration
	stats           bool

	sync.RWMutex
}

type saltResponse struct {
	Salt2Hex     string `json:"s2"`
	VersionID    int64  `json:"vid"`
	NewSalt2Hex  string `json:"new_s2"`
	NewVersionID int64  `json:"new_vid"`
}

// Version is a version number for the TapLink API
type Version int64

// String implements fmt.Stringer interface. If the version is empty, the API expects "" so this return it that way
func (v Version) String() string {
	if v == 0 {
		return fmt.Sprintf("")
	}
	return fmt.Sprintf("%d", v)
}

// Salt contains a salt for the current version, and NewSalt if a new version is available
type Salt struct {
	Salt []byte
	// VersionID is the version ID used in the request
	VersionID int64 `json:"-"`
	// NewVersionID is the new version ID to use, if any.
	NewVersionID int64 `json:"vid"`
	// NewSalt is the new salt to use if newer data pool settings are available
	NewSalt []byte `json:"-"`
}

func (s Salt) String() string {
	return hex.EncodeToString(s.Salt)
}

// VerifyPassword provides information about whether a password matched and related hashes
type VerifyPassword struct {
	Matched      bool
	VersionID    int64
	NewVersionID int64
	Hash         []byte
	NewHash      []byte
}

// String returns the hex-encoded value of the password hash
func (v VerifyPassword) String() string {
	return hex.EncodeToString(v.Hash)
}

// NewPassword returns a new password hash and the version it was created with
type NewPassword struct {
	Hash      []byte
	VersionID int64
}

// String returns the hex-encoded value of the password hash
func (p NewPassword) String() string {
	return hex.EncodeToString(p.Hash)
}

// New returns a new TapLink API connection
func New(appID string) API {
	cfg := &Config{
		appID: appID,
		host:  "https://api.taplink.co",
		headers: map[string]string{
			"User-Agent": userAgent,
			"Accept":     "application/json",
		},
	}
	return &Client{cfg: cfg}
}

// Config returns the current client configuration
func (c *Client) Config() Configuration {
	return c.cfg
}

// VerifyPassword verifies a password for an existing user which was stored using blind hashing.
// 'hash'         - hash of the user's password
// 'expected' - expected value of hash2
// 'versionId'        - version identifier for data pool settings to use
// If a new 'versionId' and 'hash2' value are returned, they can either be ignored, or both must be updated in the data store together which
// will cause the latest data pool settings to be used when blind hashing for this user in the future.
// If the versionID is 0, the default version will be used
func (c *Client) VerifyPassword(hash []byte, expected []byte, versionID int64) (*VerifyPassword, error) {
	salt, err := c.getSalt(hash, versionID)
	if err != nil {
		return nil, err
	}
	sum := hmac.New(sha512.New, salt.Salt)
	sum.Write(hash)
	vp := &VerifyPassword{Hash: sum.Sum(nil), NewVersionID: salt.NewVersionID, VersionID: salt.VersionID}
	vp.Matched = bytes.Equal(vp.Hash, expected)
	if vp.Matched && salt.VersionID != salt.NewVersionID && salt.NewSalt != nil {
		sum2 := hmac.New(sha512.New, salt.NewSalt)
		sum2.Write(hash)
		vp.NewHash = sum2.Sum(nil)
	}
	return vp, nil
}

// NewPassword calculates 'salt1' and 'hash2' for a new password, using the latest data pool settings.
// Also returns 'versionId' for the current settings, in case data pool settings are updated in the future
// Inputs:
//   'hash1Hex' - hash of the user's password, as a hex string
//   'callback' - function(err, hash2Hex, versionId)
//       o err       : 'err' from request, or null if request succeeded
//       o hash2Hex  : value of 'hash2' as a hex string
//       o versionId : version id of the current data pool settings used for this request
func (c *Client) NewPassword(hash1 []byte) (*NewPassword, error) {
	salt, err := c.getSalt(hash1, 0)
	if err != nil {
		return nil, err
	}

	// Calculate the hash of the new salt
	sum := hmac.New(sha512.New, salt.Salt)
	sum.Write(hash1)

	return &NewPassword{VersionID: salt.VersionID, Hash: sum.Sum(nil)}, nil
}

// GetSalt retreives a salt value from the data pool, given a 'hash1' value and optionally, a version id
// If requested versionId is undefined or the latest, then only a single 'salt2' value is returned with the same version id as requested
// If the requested versionId is not the latest, also returns an additional 'salt2' value along with the latest version id
// Inputs:
//    'hash1Hex'  - hex string containing value of hash1
//    'versionId' - version identifier for data pool settings to use, or 0/null/undefined to use latest settings
//    'callback'  - function(salt2Hex, versionId, newSalt2Hex, newVersionId)
//       o salt2Hex     : hex string containing value of 'salt2'
//       o versionId    : version id corresponding to the provided 'salt2Hex' value (will always match requested version, if one was specified)
//       o newSalt2Hex  : hex string containing a new value of 'salt2' if newer data pool settings are available, otherwise undefined
//       o newVersionId : a new version id, if newer data pool settings are available, otherwise undefined
func (c *Client) getSalt(hash []byte, versionID int64) (s *Salt, err error) {

	uri := fmt.Sprintf("%s/%s/%s/%s", c.Config().Host(), c.Config().AppID(), hex.EncodeToString(hash), Version(versionID))
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return
	}

	for k, v := range c.Config().Headers() {
		req.Header.Set(k, v)
	}

	var t time.Time
	var attempts int
	var resp *http.Response

	// Attempt to connect until the attempt limit has been reached.
	// Reset the timer in each loop so the final result will have the proper
	// latency value
	for {
		t = time.Now()
		resp, err = HTTPClient.Do(req)
		if err == nil || attempts > RetryLimit {
			break
		}
		if resp.TLS == nil {
			panic("Unencrypted response")
		}
		c.incrErrs(0)
		attempts++
		time.Sleep(RetryDelay)
	}

	// If failed to send the request.
	if err != nil {
		return
	}

	latency := time.Since(t)

	// Update stats regardless of what happens from here on out.
	defer func() {
		if err != nil {
			c.incrErrs(latency)
			return
		}
		c.incrSuccess(latency)
	}()

	// If request error, fail now.
	if err != nil {
		return
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// If not a 200 request, return the status text as the error message
	if resp.StatusCode != http.StatusOK {
		err = errors.New(strings.TrimSpace(string(bodyBytes)))
		return
	}

	var sr saltResponse
	err = json.Unmarshal(bodyBytes, &sr)
	if err != nil {
		return
	}

	// Use the values from the request in the return value
	s = &Salt{NewVersionID: sr.NewVersionID, VersionID: sr.VersionID}

	// Hex encoding is used over the wire, so decode here.
	s.Salt, err = hex.DecodeString(sr.Salt2Hex)
	if err != nil {
		return
	}

	if sr.NewSalt2Hex == "" {
		return
	}

	s.NewSalt, err = hex.DecodeString(sr.NewSalt2Hex)
	return
}

func (c *Client) incrErrs(latency time.Duration) {
	if !c.stats {
		return
	}
	c.Lock()
	c.reqErrCt++
	if latency != 0 {
		c.reqLatency = append(c.reqLatency, latency)
	}
	c.Unlock()
}

func (c *Client) incrSuccess(latency time.Duration) {
	if !c.stats {
		return
	}
	c.Lock()
	c.reqCt++
	c.reqLatency = append(c.reqLatency, latency)
	c.Unlock()
}
