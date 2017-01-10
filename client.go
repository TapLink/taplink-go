package taplink

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	// ensures the Client implements the API interface
	_ API = (*Client)(nil)
)

// Client is a struct which implements the API interface
type Client struct {
	cfg Configuration
	sync.RWMutex
}

// Stats returns stats about connections to the server
func (c *Client) Stats() Statistics {
	return c.cfg.Stats()
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

func (c *Client) getFromAPI(path string) (respBody []byte, err error) {

	var attempts int
	var resp *http.Response

	// Attempt to connect until the attempt limit has been reached.
	// Reset the timer in each loop so the final result will have the proper
	// latency value.
	for attempts < RetryLimit {

		// For each subsequent attempt after the first add the RetryDelay
		if attempts > 0 {
			time.Sleep(RetryDelay)
		}

		t := time.Now()
		host := c.Config().Host(attempts)

		attempts++
		urlStr := fmt.Sprintf("https://%s/%s", host, strings.TrimPrefix(path, "/"))
		req, _ := http.NewRequest("GET", urlStr, nil)
		for k, v := range c.Config().Headers() {
			req.Header.Set(k, v)
		}

		resp, err = HTTPClient.Do(req)

		// Check for a timeout, if so record it accordingly.
		netErr, isNetErr := err.(net.Error)
		urlErr, isURLErr := err.(*url.Error)
		switch {
		// Check if it's a timeout, if so record it.
		case err != nil && ((isNetErr && netErr.Timeout()) || (isURLErr && urlErr.Timeout())):
			c.Stats().AddTimeout(host)
			continue
		// For other errors, we'll add an "unknown" code since there won't
		// be any response to get the code from.
		case resp == nil:
			c.Stats().AddError(host, 999)
			continue
		}

		// If have a response to work with, get the body and determine the
		// status code. If it's non-200 then it's an error, and try again.
		latency := time.Since(t)
		defer resp.Body.Close()
		respBody, err = ioutil.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
		if err != nil || len(respBody) == 0 {
			c.Stats().AddError(host, 999)
			err = io.ErrUnexpectedEOF
			continue
		}

		switch {
		// If it's a server error, then record it and if this is the last
		// attempt, the message will be returned. Otherwise another attempt will be made.
		case resp.StatusCode >= 500:
			c.Stats().AddError(host, resp.StatusCode)
			err = errors.New(strings.TrimSpace(string(respBody)))
		// If it's a client error, then return the error, don't attempt again.
		case resp.StatusCode >= 400:
			c.Stats().AddError(host, resp.StatusCode)
			return nil, errors.New(strings.TrimSpace(string(respBody)))
		// Otherwise redirects 3xx or success 2xx are okay
		default:
			c.Stats().AddSuccess(host, latency)
			return
		}
	}

	return
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

	uri := fmt.Sprintf("%s/%s/%s", c.Config().AppID(), hex.EncodeToString(hash), Version(versionID))
	bodyBytes, err := c.getFromAPI(uri)

	// If request error, fail now.
	if err != nil {
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
