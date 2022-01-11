package types

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"sort"

	"github.com/carloszimm/github-mining/internal/config"
	"github.com/carloszimm/github-mining/internal/util"
	"github.com/dlclark/regexp2"
)

// type to store the repo name and the file content to be processed
type ContentMsg struct {
	FileName    string
	FileContent string
}

// type to store the repo name and some operator counting
type CountMsg struct {
	FileName string
	OperatorCount
}

type OperatorCount struct {
	Operator string
	Total    int
}

// type used to hold operators' names and generate workers per request
type Operators struct {
	Dist          string
	operatorsList []string
}

func (ops *Operators) GetOperators() []string {
	return ops.operatorsList
}

func (ops *Operators) CreateWorkerOps() ([]chan *ContentMsg, <-chan interface{}) {
	inChannels := make([]chan *ContentMsg, len(ops.operatorsList))
	outChannels := make([]<-chan interface{}, 0, len(ops.operatorsList)*config.PROCESSING_WORKERS)

	// ranges through the operators list and creates a channel/worker for each operator
	// stores output channels for futher merging and creation of a single channel
	for i, op := range ops.operatorsList {
		inChannel := make(chan *ContentMsg)
		inChannels[i] = inChannel
		counterFn := createCounter(op)
		for j := 0; j < config.PROCESSING_WORKERS; j++ {
			outChannels = append(outChannels, createOpWorker(inChannel, op, counterFn))
		}
	}

	return inChannels, util.MergeChannels(outChannels...)
}

func createOpWorker(in <-chan *ContentMsg, opName string, counterFn func(string) int) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		for msg := range in {
			out <- CountMsg{msg.FileName, OperatorCount{opName, counterFn(msg.FileContent)}}
		}
		close(out)
	}()
	return out
}

func CreateOperators(path string, dist string) Operators {
	data, err := ioutil.ReadFile(filepath.Join(config.OPERATORS_PATH, path))
	util.CheckError(err)

	ops := Operators{Dist: dist}

	err = json.Unmarshal(data, &ops.operatorsList)
	util.CheckError(err)

	return ops
}

// (?<!\w) - negative look-behind to make sure the operator name isn't preceded by any character beside its own name
// \s* - followed by zero or more spaces
// { - for swift closures
func createCounter(opName string) func(string) int {
	re := regexp2.MustCompile(`\.?(?<!\w)`+opName+`\s*(\(|{)`, 0)
	return func(s string) int {
		return len(util.Regexp2FindAllString(re, s))
	}
}

func CloseAllInOps(inOps []chan *ContentMsg) {
	for _, in := range inOps {
		close(in)
	}
}

// sort operator count by operators' names
func SortOperatorsCount(opCount []OperatorCount) {
	sort.SliceStable(opCount, func(i, j int) bool {
		return opCount[i].Operator < opCount[j].Operator
	})
}

// returns a map containing operator name as key and an array of total counts as value
func AggregateByOperator(result interface{}) map[string][]int {
	opsCount := make(map[string][]int)
	switch r := result.(type) {
	case map[int][]OperatorCount:
		for _, val := range r {
			for _, opCount := range val {
				opsCount[opCount.Operator] = append(opsCount[opCount.Operator], opCount.Total)
			}
		}
	case map[string][]OperatorCount:
		for _, val := range r {
			for _, opCount := range val {
				opsCount[opCount.Operator] = append(opsCount[opCount.Operator], opCount.Total)
			}
		}
	}
	return opsCount
}
