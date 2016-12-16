package examples

import (
	"log"
	"time"

	"github.com/TapLink/taplink-go"
)

func mainAlt() {

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
