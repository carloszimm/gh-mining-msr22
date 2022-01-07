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

	"github.com/carloszimm/github-mining/internal/config"
	errorHandling "github.com/carloszimm/github-mining/internal/error-handling"
	"github.com/carloszimm/github-mining/internal/types"
	"github.com/carloszimm/github-mining/internal/util"
	"github.com/golang-module/carbon/v2"
	"github.com/google/go-github/v41/github"
	"golang.org/x/oauth2"
)

var RX_USERS = map[string]struct{}{
	"ReactiveX":           {},
	"Reactive-Extensions": {},
	"dotnet":              {},
	"neuecc":              {},
	"bjornbytes":          {},
	"alfert":              {},
	"kzaher":              {},
}

const ARCHIVES_FOLDER = "archives"

var (
	REPO_SEARCH_PATH    = filepath.Join("assets", "repo-search")
	REPO_RETRIEVAL_PATH = filepath.Join("assets", "repo-retrieval")
)

type Summary struct {
	StartTime      string
	EndTime        string
	TotalRepos     int
	ProcessedRepos int
}

func setup(repos []github.Repository) <-chan *types.Info {
	cfg := config.GetConfigInstance()

	path := filepath.Join(REPO_RETRIEVAL_PATH, cfg.Distribution)
	util.RemoveAllFolders(path)
	util.WriteFolder(path)

	archivesPath := filepath.Join(path, ARCHIVES_FOLDER)

	outRepo := processRepos(repos)
	// channel used to refeed the pipeline in case of error (rate limiting)
	retroInput := make(chan github.Repository, 20)

	inWorkers := mergeChannels(outRepo, retroInput)
	outWorkers := make(chan *types.Info, 20)

	// creates workers
	for i, token := range cfg.Tokens {
		githubWorker(i, token, archivesPath, inWorkers, retroInput, outWorkers)
	}
	// creates unauthenticated worker
	githubWorker(len(cfg.Tokens), "", archivesPath, inWorkers, retroInput, outWorkers)

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

func githubWorker(id int, token string, archivesPath string, in <-chan github.Repository,
	retroInput chan github.Repository, out chan *types.Info) {
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
				errorHandling.HandleErrorWorkers(err, id, resp, client)
			} else {
				body, err := ioutil.ReadAll(resp.Body)
				util.CheckError(err)

				fileName := strings.Split(resp.Header["Content-Disposition"][0], "=")[1]

				err = os.WriteFile(filepath.Join(REPO_RETRIEVAL_PATH, archivesPath, fileName), body, 0644)
				util.CheckError(err)

				out <- &types.Info{Owner: repo.GetOwner().GetLogin(), RepositoryName: repo.GetName(),
					RepositoryFullName: repo.GetFullName(), Branch: repo.GetDefaultBranch(),
					FileName: fileName, FileSize: len(body), ArchiveUrl: repo.GetArchiveURL()}
			}
		}
	}()
}

func retrieveBranchInfoWorker(id int, token string, infos <-chan *types.Info, results chan<- types.Info) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	for i := range infos {
		for {
			branchInfo, resp, err :=
				client.Repositories.GetBranch(ctx, i.Owner, i.RepositoryName, i.Branch, true)
			if err != nil {
				errorHandling.HandleErrorWorkers(err, id, resp, client)
				continue
			}
			i.ArchiveUrl = strings.Replace(strings.Replace(i.ArchiveUrl, "{archive_format}", "tarball", 1),
				"{/ref}", "/"+branchInfo.GetCommit().GetSHA(), 1)

			results <- *i
			break
		}
	}
}

func processFileInfos(fileInfos []*types.Info) {
	cfg := config.GetConfigInstance()
	fileName := "list_of_files"

	infos := make(chan *types.Info, 20)
	results := make(chan types.Info, 20)

	// creates workers
	for i, token := range cfg.Tokens {
		go retrieveBranchInfoWorker(i, token, infos, results)
	}

	go func() {
		for _, info := range fileInfos {
			infos <- info
		}
	}()

	var newFileInfos []types.Info
	for range fileInfos {
		newFileInfos = append(newFileInfos, <-results)
	}

	util.WriteJSON(filepath.Join(REPO_RETRIEVAL_PATH, cfg.Distribution, fileName), newFileInfos)
}

func writeSummary(path string, summ *Summary) {
	template := "Start Time: %v\nEnd Time: %v\nTotal of Repositories: %v-%v\n"
	template += "Repositories Processed: %v"
	text := fmt.Sprintf(template, summ.StartTime, summ.EndTime, summ.TotalRepos, summ.ProcessedRepos)

	fileName := fmt.Sprintf("summary_%s.txt", util.NowDateTimeFormatted())

	err := os.WriteFile(filepath.Join(path, fileName), []byte(text), 0644)
	util.CheckError(err)
}

func main() {
	cfg := config.GetConfigInstance()

	c, err := os.ReadDir(REPO_SEARCH_PATH)
	util.CheckError(err)

	summ := Summary{
		StartTime: carbon.Now().ToDayDateTimeString()}

	var repos []github.Repository
	for _, entry := range c {
		// loops through folder entries and stop as soon as the entry hits the distribution being looked for
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
		summ.TotalRepos, summ.ProcessedRepos = len(repos), len(filteredRepos)

		out := setup(filteredRepos)

		// writes infos about the archives as JSON to avoid uploading all downloaded repos
		var filesInfos []*types.Info
		for range filteredRepos {
			filesInfos = append(filesInfos, <-out)
		}
		processFileInfos(filesInfos)
		// writes summary
		summ.EndTime = carbon.Now().ToDayDateTimeString()
		path := filepath.Join(REPO_RETRIEVAL_PATH, cfg.Distribution)
		writeSummary(path, &summ)

		log.Printf("Processed %d from %d repositories\n", summ.ProcessedRepos, summ.TotalRepos)
		log.Printf("Results available at: %s", path)
	} else {
		log.Println("No repositories to be processed")
	}
}
