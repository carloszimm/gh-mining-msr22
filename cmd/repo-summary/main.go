package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/carloszimm/github-mining/internal/config"
	errorhandling "github.com/carloszimm/github-mining/internal/error-handling"
	"github.com/carloszimm/github-mining/internal/util"
	"github.com/google/go-github/v41/github"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/oauth2"
)

const (
	FILE_NAME = "summary.txt"
	STARS     = 10
)

var REPO_SUMMARY_PATH = filepath.Join("assets", "repo-summary", FILE_NAME)

var DISTRIBUTIONS [18]string = [18]string{
	"RxJava", "RxJS", "Rx.NET", "UniRx", "RxScala", "RxClojure",
	"RxCpp", "RxLua", "Rx.rb", "RxPY", "RxGo", "RxGroovy", "RxJRuby",
	"RxKotlin", "RxSwift", "RxPHP", "reaxive", "RxDart"}

func starsQuery(iteration int) string {
	switch iteration {
	case 1:
		return " stars:0"
	case 2:
		return fmt.Sprintf(" stars:>=%d", STARS)
	default:
		return ""
	}
}

func retrieveRepoInfoWorker(id int, token string, queries <-chan string, results chan<- []string) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// PerPage == 1 since we want the total not the results
	opt := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 1},
	}
	for query := range queries {
		result := []string{query}
		for i := 0; i < 3; i++ {
			for {
				repos, resp, err := client.Search.Repositories(ctx, query+starsQuery(i), opt)
				if err != nil {
					errorhandling.HandleErrorWorkers(err, id, resp, client)
					continue
				}

				result = append(result, strconv.Itoa(repos.GetTotal()))
				break
			}
		}
		results <- result
	}
}

func writeData(data [][]string) {
	f, err := os.Create(REPO_SUMMARY_PATH)
	util.CheckError(err)
	defer f.Close()

	table := tablewriter.NewWriter(f)
	table.SetHeader([]string{"Distribution", "Total", "Stars = 0", fmt.Sprintf("Stars >= %d", STARS)})

	table.AppendBulk(data)

	table.Render()
}

func main() {
	cfg := config.GetConfigInstance()

	jobs := make(chan string, 3*len(cfg.Tokens))
	results := make(chan []string, 3*len(cfg.Tokens))

	// create workers according to GitHub tokens provided under config
	for i, token := range cfg.Tokens {
		go retrieveRepoInfoWorker(i, token, jobs, results)
	}

	for _, dist := range DISTRIBUTIONS {
		jobs <- dist
	}

	var queryResults [][]string
	for range DISTRIBUTIONS {
		queryResults = append(queryResults, <-results)
	}

	sort.SliceStable(queryResults, func(i, j int) bool {
		totalI, _ := strconv.Atoi(queryResults[i][1])
		totalJ, _ := strconv.Atoi(queryResults[j][1])
		return totalI > totalJ
	})

	writeData(queryResults)
}
