package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/carloszimm/github-mining/internal/config"
	"github.com/carloszimm/github-mining/internal/util"
	"github.com/golang-module/carbon/v2"
	"github.com/google/go-github/v41/github"
	"golang.org/x/oauth2"
)

type QueryOpts struct {
	Query     string
	Sort      string
	Order     string
	FirstPage bool
}

type QueryResult struct {
	Total        int
	Repositories []*github.Repository
	*QueryOpts
}

type UniqueResults struct {
	Results map[int64]*github.Repository
}

func NewUniqueResults() *UniqueResults {
	return &UniqueResults{
		Results: make(map[int64]*github.Repository),
	}
}

func (ur *UniqueResults) AddAll(repos []*github.Repository) {
	for _, repo := range repos {
		if _, ok := ur.Results[*repo.ID]; !ok {
			ur.Results[*repo.ID] = repo
		}
	}
}

func (ur *UniqueResults) Length() int {
	return len(ur.Results)
}

func (ur *UniqueResults) AsArray() []*github.Repository {
	var repos []*github.Repository
	for _, repo := range ur.Results {
		repos = append(repos, repo)
	}
	return repos
}

type Interval struct {
	diff       int64
	unit       string
	start, end carbon.Carbon
	Intervals  []string
}

func calculatePeriod(unit string, endPeriod carbon.Carbon) carbon.Carbon {
	switch unit {
	case "years":
		endPeriod = endPeriod.AddYear()
	case "months":
		endPeriod = endPeriod.AddMonth()
	case "weeks":
		endPeriod = endPeriod.AddWeek()
	case "days":
		endPeriod = endPeriod.AddDay()
	case "hours":
		endPeriod = endPeriod.AddHour()
	case "minutes":
		endPeriod = endPeriod.AddMinute()
	case "seconds":
		endPeriod = endPeriod.AddSecond()
	}
	return endPeriod
}

func (i *Interval) CalculateInterval() {
	if len(i.Intervals) == 0 {
		if i.diff > 1 {
			template := " pushed:%s..%s"
			startPeriod := i.start
			endPeriod := i.start
			for j := int64(0); j < i.diff; j++ {

				endPeriod = calculatePeriod(i.unit, endPeriod)

				if endPeriod.Gte(i.end) {
					i.Intervals = append(i.Intervals, fmt.Sprintf(template, startPeriod.ToIso8601String(), i.end.ToIso8601String()))
					break
				}
				if j+1 >= i.diff { // last loop execution
					if endPeriod.Lt(i.end) {
						i.Intervals = append(i.Intervals, fmt.Sprintf(template, endPeriod.ToIso8601String(), i.end.ToIso8601String()))
						break
					}
				}
				i.Intervals = append(i.Intervals, fmt.Sprintf(template, startPeriod.ToIso8601String(), endPeriod.ToIso8601String()))
				startPeriod = endPeriod
			}
		}
	}
}

func (i *Interval) Ok() bool {
	return i.diff > 1
}

type DifferenceIterator struct {
	index     int
	intervals []*Interval
}

func NewDifferenceIterator(start, end carbon.Carbon) *DifferenceIterator {
	iterator := &DifferenceIterator{}
	timeUnits := []string{"years", "months", "weeks", "days", "hours", "minutes", "seconds"}

	for _, unit := range timeUnits {
		switch unit {
		case "years":
			i := &Interval{diff: end.DiffInYearsWithAbs(start), unit: "years", start: start, end: end}
			iterator.intervals = append(iterator.intervals, i)
		case "months":
			i := &Interval{diff: end.DiffInMonthsWithAbs(start), unit: "months", start: start, end: end}
			iterator.intervals = append(iterator.intervals, i)
		case "weeks":
			i := &Interval{diff: end.DiffInWeeksWithAbs(start), unit: "weeks", start: start, end: end}
			iterator.intervals = append(iterator.intervals, i)
		case "days":
			i := &Interval{diff: end.DiffInDaysWithAbs(start), unit: "days", start: start, end: end}
			iterator.intervals = append(iterator.intervals, i)
		case "hours":
			i := &Interval{diff: end.DiffInHoursWithAbs(start), unit: "hours", start: start, end: end}
			iterator.intervals = append(iterator.intervals, i)
		case "minutes":
			i := &Interval{diff: end.DiffInMinutesWithAbs(start), unit: "minutes", start: start, end: end}
			iterator.intervals = append(iterator.intervals, i)
		case "seconds":
			i := &Interval{diff: end.DiffInSecondsWithAbs(start), unit: "seconds", start: start, end: end}
			iterator.intervals = append(iterator.intervals, i)
		}
	}
	log.Println(iterator.intervals)
	return iterator
}

func (u *DifferenceIterator) hasNext() bool {
	return u.index < len(u.intervals)

}
func (u *DifferenceIterator) getNext() *Interval {
	if u.hasNext() {
		intervals := u.intervals[u.index]
		u.index++
		return intervals
	}
	return nil
}

func constructStarInterval(minStars, maxStars, factor int) []string {
	var interval []string
	for ; minStars < maxStars; minStars += factor {
		diff := maxStars - minStars
		if diff < factor {
			factor = diff
		}
		interval = append(interval, fmt.Sprintf(" stars:%d..%d", minStars, minStars+factor))
	}
	return interval
}

func handleStarSearch(jobs chan *QueryOpts, uniqueResults *UniqueResults,
	query string, total int, intervals []string) {

	for i := 0; i < len(intervals) && uniqueResults.Length() < total; i++ {
		jobs <- &QueryOpts{
			Query: query + intervals[i],
		}
	}
}

func handleTimeIntervalsSearch(jobs chan *QueryOpts, results chan *QueryResult,
	uniqueResults *UniqueResults, total int, queries []string) {
	for _, queryStars := range queries {
		orders := [2]string{"asc", "desc"}
		var start, end carbon.Carbon
		for _, order := range orders {
			jobs <- &QueryOpts{
				Query:     queryStars,
				Sort:      "updated",
				Order:     order,
				FirstPage: true,
			}
		}
		for i := range orders {
			resultTimes := <-results
			if i == 0 {
				start = carbon.Time2Carbon(resultTimes.Repositories[0].UpdatedAt.Time)
			} else {
				end = carbon.Time2Carbon(resultTimes.Repositories[0].UpdatedAt.Time)
			}
		}
		if start.Gt(end) {
			start, end = end, start
		}

		iterator := NewDifferenceIterator(start, end)

		for uniqueResults.Length() < total && iterator.hasNext() {
			i := iterator.getNext()
			if i.Ok() { //this interval greater than 1
				i.CalculateInterval() //calculate interval as required (lazy evaluation)
				countResult := 0
				for j, interval := range i.Intervals {
					jobs <- &QueryOpts{
						Query: queryStars + interval,
					}
					if j != 0 && j%3 == 0 {
						for k := 0; k < 3; k++ {
							r := <-results
							uniqueResults.AddAll(r.Repositories)
							countResult++
						}
						// breaks earlier if total result already obtained
						if uniqueResults.Length() >= total {
							continue
						}
					}
				}
				if countResult < len(i.Intervals) { //if there are still results
					for ; countResult < len(i.Intervals); countResult++ {
						r := <-results
						uniqueResults.AddAll(r.Repositories)
					}
				}
			}
		}
	}

}

func main() {
	cfg := *config.GetConfigInstance()

	jobs := make(chan *QueryOpts, 3*len(cfg.Tokens))
	results := make(chan *QueryResult, 3*len(cfg.Tokens))

	// create workers according to GitHub tokens provided under config
	for w := 0; w < len(cfg.Tokens); w++ {
		go worker(w, cfg.Tokens[w], jobs, results)
	}

	log.Printf("Starting search for %s\n", cfg.Distribution)

	jobs <- &QueryOpts{
		Query: fmt.Sprintf("%s stars:>=%d", cfg.Distribution, cfg.MinStars),
		Sort:  "stars",
		Order: "desc",
	}
	result := <-results

	if result.Total > 1000 {
		var excedingQueries []string
		uniqueResults := NewUniqueResults()
		uniqueResults.AddAll(result.Repositories)

		intervals := constructStarInterval(cfg.MinStars,
			result.Repositories[0].GetStargazersCount(), cfg.IncreaseFactor)

		go handleStarSearch(jobs, uniqueResults, cfg.Distribution, result.Total, intervals)

		for i := 0; i < len(intervals) && uniqueResults.Length() < result.Total; i++ {
			starResult := <-results
			if starResult.Total > 1000 { //store it for further investigation
				excedingQueries = append(excedingQueries, starResult.Query)
			}
			uniqueResults.AddAll(starResult.Repositories)
		}

		// check if there are still results to be computed
		// handle them by breaking the queries in subqueries with time intervals
		if uniqueResults.Length() < result.Total && len(excedingQueries) > 0 {
			handleTimeIntervalsSearch(jobs, results, uniqueResults, result.Total, excedingQueries)
		}

		// if there are still results after all, rerun the initial query if time interval periods
		if uniqueResults.Length() < result.Total {
			handleTimeIntervalsSearch(jobs, results, uniqueResults, result.Total,
				[]string{fmt.Sprintf("%s stars:>=%d", cfg.Distribution, cfg.MinStars)})
		}

		result.Repositories = uniqueResults.AsArray()
	}
	log.Printf("Total results: %d, Results retrieved: %d\n", result.Total, len(result.Repositories))
	log.Println("Writing results...")
	util.WriteJSON(filepath.Join("assets", "repo-search",
		cfg.Distribution+"_"+strings.ReplaceAll(carbon.Now().ToDateTimeString(), ":", "-")),
		result.Repositories)
}

func worker(id int, token string, jobs <-chan *QueryOpts, results chan<- *QueryResult) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	opt := &github.SearchOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for j := range jobs {
		if j.Sort != "" {
			opt.Sort = j.Sort
		}
		if j.Order != "" {
			opt.Order = j.Order
		}
		queryResult := &QueryResult{QueryOpts: j}

		for { //handle pages
			log.Println("worker:", id, "query:", j.Query)
			repos, resp, err := client.Search.Repositories(ctx, j.Query, opt)

			if err != nil {
				if _, ok := err.(*github.RateLimitError); ok {
					d := resp.Rate.Reset.Time.Sub(time.Now())
					log.Println("worker", id, "went to sleep for", fmt.Sprint(d.Minutes()), "minutes")
					time.Sleep(d) //sleep and reexecute the same query again
					continue
				} else {
					log.Fatal(err)
				}
			}
			queryResult.Repositories = append(queryResult.Repositories, repos.Repositories...)
			if j.FirstPage {
				break
			}
			if resp.NextPage == 0 { //no more pages
				queryResult.Total = *repos.Total
				break
			}
			opt.Page = resp.NextPage
		}
		results <- queryResult
	}
}
