package taplink

import (
	"encoding/hex"
	"errors"
	"fmt"
	"time"
)

// Host selection algorithms
const (
	HostSelectRandom     = iota
	HostSelectRoundRobin = iota
)

var (

	// DefaultTimeout is the default HTTP request timeout
	DefaultTimeout = 30 * time.Second
	// DefaultKeepAlive is the default HTTP keep-alive duration
	DefaultKeepAlive = 30 * time.Second

	// RetryLimit indicates how many times a connection should be retried before failing
	RetryLimit = 3
	// RetryDelay is the duration to wait between retry attempts
	RetryDelay = 1 * time.Second

	// maxResponseSize is the largest Content-Length allowed from the API
	// prevents consuming too much memory from overly large upstream responses
	// that should theoretically never be the case, but it's there just in case
	maxResponseSize int64 = 1024 * 500

	// ErrHostNotFound is returned if the given host does not exist
	ErrHostNotFound = errors.New("host not found")
)

// API is an interface which exposes TapLink API functionality
type API interface {

	// Config
	Config() Configuration

	// API funcs
	VerifyPassword(hash []byte, expectedHash []byte, versionID int64) (*VerifyPassword, error)
	NewPassword(hash []byte) (*NewPassword, error)

	// Stats returns stats about each host the client has connected to
	Stats() Statistics
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
		stats: newStatistics(),
		headers: map[string]string{
			"User-Agent": userAgent,
			"Accept":     "application/json",
		},
	}
	return &Client{cfg: cfg}
}
