package processing

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/carloszimm/github-mining/internal/config"
	"github.com/carloszimm/github-mining/internal/types"
	"github.com/carloszimm/github-mining/internal/util"
	"github.com/dlclark/regexp2"
	"github.com/iancoleman/orderedmap"
)

var (
	commentsReg = regexp2.MustCompile(
		`((?:(?:^[ \t]*)?(?:/\*[^*]*\*+(?:[^/*][^*]*\*+)*/(?:[ \t]*\r?\n(?=[ \t]*(?:\r?\n|/\*|//)))?|//(?:[^\\]|\\(?:\r?\n)?)*?(?:\r?\n(?=[ \t]*(?:\r?\n|/\*|//))|(?=\r?\n))))+)|("[^"\\]*(?:\\[\S\s][^"\\]*)*"|'[^'\\]*(?:\\[\S\s][^'\\]*)*'|(?:\r?\n|[\S\s])[^/"'\\\s]*)`, 0)
	stringsReg = regexp2.MustCompile(
		`(["'`+"`"+`])(?:(?=(\\?))\2.)*?\1`, 0)
)

func SetupOpsPipeline( /* archInfosMap map[string]string, */
	allowedExtensions map[string]struct{}, operators types.Operators,
	result map[string]*orderedmap.OrderedMap) <-chan map[string]*orderedmap.OrderedMap {
	// create workers from the list of operators
	inOps, outOps := operators.CreateWorkerOps()

	out := processArchives(operators.Dist)

	var outChannels []<-chan interface{}
	for i := 0; i < config.PROCESSING_WORKERS; i++ {
		outChannels = append(outChannels, processArchive(out, allowedExtensions, operators.Dist))
	}
	out = util.MergeChannels(outChannels...)

	outChannels = nil
	for i := 0; i < config.PROCESSING_WORKERS; i++ {
		outChannels = append(outChannels, removeComments(out))
	}
	out = util.MergeChannels(outChannels...)

	outChannels = nil
	for i := 0; i < config.PROCESSING_WORKERS; i++ {
		outChannels = append(outChannels, removeStrings(out))
	}
	out = util.MergeChannels(outChannels...)

	// (?i) case insensitive
	reg := regexp.MustCompile("(?i)" + operators.Dist)
	outChannels = nil
	for i := 0; i < config.PROCESSING_WORKERS; i++ {
		outChannels = append(outChannels, checkImport(out, reg))
	}
	out = util.MergeChannels(outChannels...)

	dispatchToOpsCounters(out, inOps)

	return gatherResults(outOps, len(operators.GetOperators()), result)
}

func processArchives(dist string) <-chan interface{} {
	out := make(chan interface{})

	archives, err := os.ReadDir(filepath.Join(config.REPO_RETRIVAL_PATH, dist, config.ARCHIVES_FOLDER))
	util.CheckError(err)

	go func() {
		for _, val := range archives {
			out <- val
		}
		close(out)
	}()
	return out
}

func processArchive(in <-chan interface{},
	allowedExtensions map[string]struct{}, dist string) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		for val := range in {
			entry := val.(fs.DirEntry)

			file, err := os.Open(filepath.Join(config.REPO_RETRIVAL_PATH, dist,
				config.ARCHIVES_FOLDER, entry.Name()))
			util.CheckError(err)

			archive, err := gzip.NewReader(file)
			util.CheckError(err)

			tr := tar.NewReader(archive)

			for {
				hdr, err := tr.Next()
				if err == io.EOF { //no more files to process
					break
				}
				util.CheckError(err)

				// regular file
				if hdr.Typeflag == tar.TypeReg {
					// check if it is in the list of allowed extensions before reading its content
					if _, ok := allowedExtensions[filepath.Ext(hdr.Name)]; ok {
						//uncomment to see info about the file being processed
						//log.Printf("Processing file %s from %s\n", hdr.Name, entry.Name())
						bs, err := ioutil.ReadAll(tr)
						if err != nil { // check for errors
							log.Printf("Repository:%s, File:%s\n", entry.Name(), hdr.Name)
							log.Fatal(err)
						}
						fileContent := string(bs)
						//uncomment it to check files' content
						//log.Println(fileContent)
						out <- &types.ContentMsg{FileName: entry.Name(), FileContent: fileContent}
					}
				}
			}
			file.Close()
		}
		close(out)
	}()
	return out
}

func removeComments(in <-chan interface{}) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		for msg := range in {
			t := msg.(*types.ContentMsg)
			// loops through the matches, replacing them by space
			for _, result := range util.Regexp2FindAllString(commentsReg, t.FileContent) {
				t.FileContent = strings.Replace(t.FileContent, result[1].String(), " ", 1)
			}
			out <- t
		}
		close(out)
	}()
	return out
}

func removeStrings(in <-chan interface{}) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		for msg := range in {
			t := msg.(*types.ContentMsg)
			results := util.Regexp2FindAllString(stringsReg, t.FileContent)
			// loops through the matches, replacing them by empty string
			for _, result := range results {
				t.FileContent = strings.Replace(t.FileContent, result[0].String(), "", 1)
			}
			out <- t
		}
		close(out)
	}()
	return out
}

func checkImport(in <-chan interface{}, reg *regexp.Regexp) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		for msg := range in {
			t := msg.(*types.ContentMsg)
			if reg.MatchString(t.FileContent) {
				out <- t
			}
		}
		close(out)
	}()
	return out
}

// broadcast to operator counters(workers)
func dispatchToOpsCounters(in <-chan interface{}, inOps []chan *types.ContentMsg) {
	go func() {
		for msg := range in {
			t := msg.(*types.ContentMsg)
			for _, inOp := range inOps {
				inOp <- t
			}
		}
		// closes all channels when done
		types.CloseAllInOps(inOps)
	}()
}

func gatherResults(outOps <-chan interface{},
	maxOperators int, result map[string]*orderedmap.OrderedMap) <-chan map[string]*orderedmap.OrderedMap {
	out := make(chan map[string]*orderedmap.OrderedMap)
	go func() {
		for msg := range outOps {
			countMsg := msg.(types.CountMsg)
			countI, _ := result[countMsg.FileName].Get(countMsg.OperatorCount.Operator)
			count := countI.(int)
			result[countMsg.FileName].Set(countMsg.OperatorCount.Operator,
				count+countMsg.OperatorCount.Total)
		}
		log.Println("General processing finished!")
		// sort results by operators' names
		for _, val := range result {
			val.SortKeys(sort.Strings)
		}
		out <- result
		close(out)
	}()
	return out
}
