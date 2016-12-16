// +build appengine

package taplink

import (
	"google.golang.org/appengine"
	"google.golang.org/appengine/urlfetch"
)

var (
	goVersion = appengine.InstanceID()

	HTTPClient = urlfetch.New(appengine.BackgroundContext())
)
