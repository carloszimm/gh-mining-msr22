package errorhandling

import (
	"context"
	"log"
	"time"

	"github.com/google/go-github/v41/github"
)

func HandleErrorWorkers(err error, id int, resp *github.Response, client *github.Client) {
	ctx := context.Background()

	if _, ok := err.(*github.RateLimitError); ok {
		handleSleep(id, time.Until(resp.Rate.Reset.Time))
	} else {
		// checks the Rate Limiting API in case the above doesn't work properly
		rateLimit, _, errRLimit := client.RateLimits(ctx)
		if err == nil {
			limit := rateLimit.GetCore()
			if limit.Remaining == 0 {
				handleSleep(id, time.Until(limit.Reset.Time))
			} else {
				// search API has a different limit
				limit = rateLimit.GetSearch()
				if limit.Remaining == 0 {
					handleSleep(id, time.Until(limit.Reset.Time))
				}
			}
		} else {
			// log errors
			log.Println(err)
			log.Println(errRLimit)
		}
	}
}

func handleSleep(id int, d time.Duration) {
	log.Printf("worker %d went to sleep for %f minutes", id, d.Minutes())
	time.Sleep(d)
}
