// +build !appengine

package taplink

import (
	"net"
	"net/http"
	"runtime"
)

var (
	goVersion = runtime.Version()

	// HTTPClient defines the HTTP client used for HTTP connections
	HTTPClient = &http.Client{
		Timeout: DefaultTimeout,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   DefaultTimeout,
				KeepAlive: DefaultKeepAlive,
			}).Dial,
		},
	}
)
