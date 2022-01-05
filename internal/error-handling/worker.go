package errorhandling

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/go-github/v41/github"
)

func HandleErrorWorkers(err error, id int, resp *github.Response, client *github.Client) {
	ctx := context.Background()

	if _, ok := err.(*github.RateLimitError); ok {
		d := time.Until(resp.Rate.Reset.Time)
		log.Println("worker", id, "went to sleep for", fmt.Sprint(d.Minutes()), "minutes")
		time.Sleep(d)
	} else {
		// checks the Rate Limiting API in case the above doesn't work properly
		rateLimit, _, errRLimit := client.RateLimits(ctx)
		if err == nil {
			coreLimit := rateLimit.GetCore()
			if coreLimit.Remaining == 0 {
				d := time.Until(coreLimit.Reset.Time)
				log.Println("worker", id, "went to sleep for", fmt.Sprint(d.Minutes()), "minutes")
				time.Sleep(d)
			}
		} else {
			// log errors
			log.Println(err)
			log.Println(errRLimit)
		}
	}
}
