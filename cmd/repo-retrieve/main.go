package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
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

type RepoInfo struct {
	Repository github.Repository
	Entry      *github.TreeEntry
	Blob       *github.Blob
}

var FILE_EXTENSIONS = map[string]map[string]struct{}{
	"rxjava": {
		".java": struct{}{}},
	"rxjs": {
		".cs": struct{}{}, ".ts": struct{}{}, ".js": struct{}{}},
	"rxkotlin": {
		".kt": struct{}{}, ".java": struct{}{}},
	"rxswift": {".swift": struct{}{}},
}
var RX_USERS = map[string]struct{}{
	"ReactiveX": {}, "dotnet": {},
	"neuecc": {}, "bjornbytes": {}, "alfert": {},
	"Reactive-Extensions": {}}

func setupPipeline(repos []github.Repository) <-chan struct{} {
	cfg := config.GetConfigInstance()

	util.RemoveAllFolders(filepath.Join("assets", "repo-retrieval", cfg.Distribution))

	outRepo := processRepositories(repos)

	var outsRepoInfo []<-chan *RepoInfo
	for i, token := range cfg.Tokens {
		outsRepoInfo = append(outsRepoInfo, githuWorker(i, token, cfg.Distribution, outRepo))
	}

	var finalOutputs []<-chan struct{}
	for _, outRepoInfo := range outsRepoInfo {
		for j := 0; j < 10; j++ {
			finalOutputs = append(finalOutputs, processContent(cfg.Distribution, outRepoInfo))
		}
	}
	return mergeChannels(finalOutputs...)
}

// based on https://go.dev/blog/pipelines
func mergeChannels(cs ...<-chan struct{}) <-chan struct{} {
	var wg sync.WaitGroup
	out := make(chan struct{})

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan struct{}) {
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

func processRepositories(repos []github.Repository) <-chan github.Repository {
	out := make(chan github.Repository, 9)
	go func() {
		for _, repo := range repos {
			out <- repo
		}
		close(out)
	}()
	return out
}

func githuWorker(id int, token string, dist string, in <-chan github.Repository) <-chan *RepoInfo {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	out := make(chan *RepoInfo, 10)
	go func() {
		for repo := range in {
			if _, ok := RX_USERS[repo.GetOwner().GetLogin()]; !ok {
				log.Printf("Worker %d processing %s\n", id, repo.GetFullName())
				var repoTree *github.Tree
				for {
					tree, resp, err := client.Git.GetTree(ctx,
						repo.GetOwner().GetLogin(), repo.GetName(), repo.GetDefaultBranch(), true)
					if err != nil {
						if _, ok := err.(*github.RateLimitError); ok {
							d := time.Until(resp.Rate.Reset.Time)
							log.Println("worker", id, "went to sleep for", fmt.Sprint(d.Minutes()), "minutes")
							time.Sleep(d)
							continue
						} else {
							log.Fatal(err)
						}
					}
					repoTree = tree
					break
				}
				for _, entry := range repoTree.Entries {
					if entry.GetType() == "blob" {
						fileExtension := filepath.Ext(entry.GetPath())
						if _, ok := FILE_EXTENSIONS[dist][fileExtension]; ok {
							for {
								blob, resp, err := client.Git.GetBlob(ctx, repo.GetOwner().GetLogin(), repo.GetName(), entry.GetSHA())
								if err != nil {
									if _, ok := err.(*github.RateLimitError); ok {
										d := time.Until(resp.Rate.Reset.Time)
										log.Println("worker", id, "went to sleep for", fmt.Sprint(d.Minutes()), "minutes")
										time.Sleep(d)
										continue
									} else {
										log.Fatal(err)
									}
								}
								out <- &RepoInfo{Repository: repo, Entry: entry, Blob: blob}
								break
							}
						}
					}
				}

			}
		}
		close(out)
	}()
	return out
}

func processContent(dist string, in <-chan *RepoInfo) <-chan struct{} {
	out := make(chan struct{})
	go func() {
		for repoInfo := range in {
			sDec, err := base64.StdEncoding.DecodeString(string(repoInfo.Blob.GetContent()))
			util.CheckError(err)

			basePath := filepath.Join("assets", "repo-retrieval", dist,
				fmt.Sprintf("%s_%d", repoInfo.Repository.GetName(), repoInfo.Repository.GetID()))
			util.WriteFolder(filepath.Join(basePath, filepath.Dir(repoInfo.Entry.GetPath())))

			err = os.WriteFile(filepath.Join(basePath, repoInfo.Entry.GetPath()), sDec, 0644)
			util.CheckError(err)

			out <- struct{}{}
		}
		close(out)
	}()
	return out
}

func main() {
	cfg := config.GetConfigInstance()

	basePath := filepath.Join("assets", "repo-search")
	c, err := os.ReadDir(basePath)
	util.CheckError(err)

	var repos []github.Repository
	for _, entry := range c {
		if !entry.IsDir() && strings.Split(entry.Name(), "_")[0] == cfg.Distribution {
			dat, err := os.ReadFile(filepath.Join(basePath, entry.Name()))
			util.CheckError(err)

			err = json.Unmarshal(dat, &repos)
			util.CheckError(err)
			break
		}
	}

	if len(repos) > 0 {
		out := setupPipeline(repos)
		counter := 0
		for range out {
			counter++
		}
		log.Printf("Processed %d repositories\n", counter)
	} else {
		log.Println("No repositories to be processed")
	}
}
