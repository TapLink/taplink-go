package examples

import (
	"log"
	"time"

	"github.com/bradberger/taplink-go"
)

func mainAlt() {

	// You can update the RetryLimit and RetryDelay for failed HTTP requests, too.
	// The API client will adhere to these settings.
	taplink.RetryLimit = 10
	taplink.RetryDelay = 30 * time.Second

	api := taplink.New("my-api-key")

	// To enable the collection of stats for the API client, use Stats().Enable()
	// By default the stats are disabled.
	api.Stats().Enable()
	api.VerifyPassword([]byte("my-password-hash"), []byte("expected"), 0)

	// If you want to load config from the TapLink api and use servers other than the the taplink.DefaultHost, then load config
	if err := api.Config().Load(); err != nil {
		log.Println("couldn't load config", err)
	}

	// After loading config, you can access the list of servers the client can connect to with Config().Servers
	log.Println("using servers", api.Config().Servers())

	// To change the connection strategy to use a random server:
	taplink.HostSelectionMethod = taplink.HostSelectRandom

	// To change the connection strategy to use a round robin selection stragegy:
	taplink.HostSelectionMethod = taplink.HostSelectRoundRobin

	// To get the stats, use these funcs...
	log.Println("total number of requests made", api.Stats().Get(taplink.DefaultHost).Requests())
	log.Println("history of latency for each successful request", api.Stats().Get(taplink.DefaultHost).Latency())
	log.Println("average time of requests", api.Stats().Get(taplink.DefaultHost).Latency().Avg())
	log.Println("num requests which had errors", api.Stats().Get(taplink.DefaultHost).Errors())

	// To disable the collection of stats, use DisableStats()
	api.Stats().Disable()
}
