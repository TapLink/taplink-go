// +build appengine

package taplink

import (
	"golang.org/x/net/context"
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

var (
	goVersion = appengine.InstanceID()

	// HTTPClient is the default HTTP client to use for requests. This won't
	// work directly in App Engine, as it's an invalid context. But at least it
	// won't panic. Use UseContext() to set a valid context before making
	// any HTTP requests.
	HTTPClient = urlfetch.New(appengine.BackgroundContext())
)

// UseContext updates the underlying HTTP client to an App Engine valid HTTP
// client which uses the given context. The HTTPClient is the result of a
// urlfetch.New() call.
func UseContext(ctx context.Context) {
	HTTPClient = urlfetch.New(ctx)
}
