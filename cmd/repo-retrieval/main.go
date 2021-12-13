package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/carloszimm/github-mining/internal/config"
	"github.com/carloszimm/github-mining/internal/util"
	"github.com/google/go-github/v41/github"
	"golang.org/x/oauth2"
)

var RX_USERS = map[string]struct{}{
	"ReactiveX": {}, "dotnet": {},
	"neuecc": {}, "bjornbytes": {}, "alfert": {},
	"Reactive-Extensions": {}}

var (
	REPO_SEARCH_PATH    = filepath.Join("assets", "repo-search")
	REPO_RETRIEVAL_PATH = filepath.Join("assets", "repo-retrieval")
)

func setup(repos []github.Repository) <-chan struct{} {
	cfg := config.GetConfigInstance()

	path := filepath.Join(REPO_RETRIEVAL_PATH, cfg.Distribution)
	util.RemoveAllFolders(path)
	util.WriteFolder(path)

	outRepo := processRepos(repos)
	retroInput := make(chan github.Repository, 20)

	inWorkers := mergeChannels(outRepo, retroInput)
	outWorkers := make(chan struct{}, 20)

	for i, token := range cfg.Tokens {
		githubWorker(i, token, cfg.Distribution, inWorkers, retroInput, outWorkers)
	}
	// creates unauthenticated worker
	githubWorker(len(cfg.Tokens), "", cfg.Distribution, inWorkers, retroInput, outWorkers)

	return outWorkers

}

func processRepos(repos []github.Repository) chan github.Repository {
	out := make(chan github.Repository, 9)
	go func() {
		for _, repo := range repos {
			out <- repo
		}
		close(out)
	}()
	return out
}

// based on https://go.dev/blog/pipelines
func mergeChannels(cs ...chan github.Repository) <-chan github.Repository {
	var wg sync.WaitGroup
	out := make(chan github.Repository)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan github.Repository) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func githubWorker(id int, token string, dist string, in <-chan github.Repository,
	retroInput chan github.Repository, out chan struct{}) {
	ctx := context.Background()

	var tc *http.Client
	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc = oauth2.NewClient(ctx, ts)
	}

	client := github.NewClient(tc)

	go func() {
		for repo := range in {
			log.Printf("GitHub Worker %d processing %s\n", id, repo.GetFullName())

			req, err := client.NewRequest("GET",
				fmt.Sprintf("repos/%s/%s/tarball", repo.GetOwner().GetLogin(), repo.GetName()), nil)
			util.CheckError(err)

			resp, err := client.BareDo(ctx, req)

			if err != nil {
				//refeeds the pipeline
				go func() {
					retroInput <- repo
				}()
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
			} else {
				body, err := ioutil.ReadAll(resp.Body)
				util.CheckError(err)

				basePath := filepath.Join(REPO_RETRIEVAL_PATH, dist,
					strings.Split(resp.Header["Content-Disposition"][0], "=")[1])

				err = os.WriteFile(basePath, body, 0644)
				util.CheckError(err)

				out <- struct{}{}
			}
		}
	}()
}

func main() {
	cfg := config.GetConfigInstance()

	c, err := os.ReadDir(REPO_SEARCH_PATH)
	util.CheckError(err)

	var repos []github.Repository
	for _, entry := range c {
		if !entry.IsDir() && strings.Split(entry.Name(), "_")[0] == cfg.Distribution {
			dat, err := os.ReadFile(filepath.Join(REPO_SEARCH_PATH, entry.Name()))
			util.CheckError(err)

			err = json.Unmarshal(dat, &repos)
			util.CheckError(err)

			break
		}
	}

	if len(repos) > 0 {
		var filteredRepos []github.Repository
		for _, repo := range repos {
			if _, ok := RX_USERS[repo.GetOwner().GetLogin()]; !ok {
				filteredRepos = append(filteredRepos, repo)
			}
		}
		out := setup(filteredRepos)

		for i := 0; i < len(filteredRepos); i++ {
			<-out
		}

		log.Printf("Processed %d from %d repositories\n", len(filteredRepos), len(repos))
	} else {
		log.Println("No repositories to be processed")
	}
}
