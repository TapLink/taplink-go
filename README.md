## Usage

Basic usage is as follows:

```go
import (
    "log"

    "github.com/TapLink/go-taplink"
)

func main() {

    api := taplink.New("my-api-key")
    pwd, err := api.NewPassword([]byte("my-password-hash"))
    if err != nil {
        log.Println("NewPassword error", err)
        return
    }

    verify, err := api.VerifyPassword([]byte("my-password-hash"), pwd.Hash, pwd.VersionID)
    if err != nil {
        log.Println("VerifyPassword error", err)
        return
    }

    log.Println("Did it match?", verify.Matched)
}
```

You can also set parameters related to HTTP requests, and also enable/disable
tracking of statistics:

```go
import (
	"log"
	"time"

	"github.com/TapLink/taplink-go"
)

func main() {

	// You can update the RetryLimit and RetryDelay for failed HTTP requests, too.
	// The API client will adhere to these settings.
	taplink.RetryLimit = 10
	taplink.RetryDelay = 30 * time.Second

	api := taplink.New("my-api-key")

	// To enable the collection of stats for the API client, use EnableStats()
	api.EnableStats()
	api.VerifyPassword([]byte("my-password-hash"), []byte("expected"), 0)

	// To get the stats, use these funcs...
	log.Println("total number of requests made", api.Requests())
	log.Println("num requests which had errors", api.Errors())
	log.Println("pct of requests which had errors", api.ErrorPct())
	log.Println("average time of requests", api.Latency())

	// To disable the collection of stats, use DisableStats()
	api.DisableStats()
}
```

If you're using on App Engine, then you'll need to set the HTTPClient with a valid
App Engine compatible HTTP client. You'll have to do this for every request.
You can do this in two ways:

```go
import (
    "net/http"

    "google.golang.org/appengine"

    "github.com/TapLink/taplink-go"
)

func myHandler(w http.ResponseWriter, r *http.Request) {

    ctx := appengine.NewContext(r)

    // First option, set the context with UseContext(). Note that this function
    // is not available for code which is not running in App Engine, and won't
    // compile outside the App Engine environment.
    taplink.UseContext(ctx)

    // Second option, set the HTTPClient directly. This would allow further
    // customization of the client if needed.
    client := urlfetch.New(ctx)
    taplink.HTTPClient = client

    // Now do something with the Taplink library, as in previous examples...
}

```
