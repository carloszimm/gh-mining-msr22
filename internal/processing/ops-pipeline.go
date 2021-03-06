package processing

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
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

var CheckFalsePositives bool

// comment pattern acquired from:
// https://stackoverflow.com/questions/36725194/golang-regex-replace-excluding-quoted-strings

const (
	commentsPattern = `((?:(?:^[ \t]*)?(?:/\*[^*]*\*+(?:[^/*][^*]*\*+)*/(?:[ \t]*\r?\n(?=[ \t]*(?:\r?\n|/\*|//)))?|//(?:[^\\]|\\(?:\r?\n)?)*?(?:\r?\n(?=[ \t]*(?:\r?\n|/\*|//))|(?=\r?\n))))+)|("[^"\\]*(?:\\[\S\s][^"\\]*)*"|'[^'\\]*(?:\\[\S\s][^'\\]*)*'|(?:\r?\n|[\S\s])[^/"'\\\s]*)`
	stringsPattern  = `(["'` + "`" + `])(?:(?=(\\?))\2.)*?\1`
)

var stringsReg = regexp2.MustCompile(stringsPattern, 0)

func SetupOpsPipeline(allowedExtensions map[string]struct{}, operators *types.Operators,
	result *orderedmap.OrderedMap) <-chan int {
	// create workers from the list of operators
	inOps, outOps := operators.CreateWorkerOps()

	out := processArchives(operators.Dist)

	var i int
	outChannels := make([]<-chan interface{}, config.PROCESSING_WORKERS)
	for i = 0; i < config.PROCESSING_WORKERS; i++ {
		outChannels[i] = processArchive(out, allowedExtensions, operators.Dist)
	}
	out = util.MergeChannels(outChannels...)

	// pass a different regex for each goroutine to avoid possible contention
	// given the complexity of the regular expression
	for i = 0; i < config.PROCESSING_WORKERS; i++ {
		outChannels[i] = removeComments(out, regexp2.MustCompile(commentsPattern, 0))
	}
	out = util.MergeChannels(outChannels...)

	// check imports before removing strings to avoid not matching
	// string paths of the imports (JS)
	regDistPattern := "(?i)" + operators.Dist
	if strings.EqualFold(operators.Dist, "RxJava") {
		regDistPattern += "|reactivex"
	}
	regDist := regexp.MustCompile(regDistPattern)
	for i = 0; i < config.PROCESSING_WORKERS; i++ {
		outChannels[i] = checkImport(out, regDist)
	}
	out = util.MergeChannels(outChannels...)

	for i = 0; i < config.PROCESSING_WORKERS; i++ {
		outChannels[i] = removeStrings(out)
	}
	out = util.MergeChannels(outChannels...)

	dispatchToOpsCounters(out, inOps)

	return gatherResults(outOps, result)
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
						//uncomment it to see info about the file being processed
						//log.Printf("Processing file %s from %s\n", hdr.Name, entry.Name())
						bs, err := ioutil.ReadAll(tr)
						if err != nil { // check for errors
							log.Printf("Repository:%s, File:%s\n", entry.Name(), hdr.Name)
							log.Fatal(err)
						}

						//uncomment it to check file's content
						//log.Println(string(bs))
						out <- types.ContentMsg{FileName: entry.Name(),
							InnerFileName: hdr.Name, FileContent: string(bs)}
					}
				}
			}
			file.Close()
		}
		close(out)
	}()
	return out
}

func removeComments(in <-chan interface{}, commentsReg *regexp2.Regexp) <-chan interface{} {
	out := make(chan interface{})
	go func() {
		for msg := range in {
			t := msg.(types.ContentMsg)
			// replace comments by space(s)
			t.FileContent, _ = commentsReg.Replace(t.FileContent, "$2 ", -1, -1)
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
			t := msg.(types.ContentMsg)
			// replace strings by empty string
			t.FileContent, _ = stringsReg.Replace(t.FileContent, "", -1, -1)
			out <- t
		}
		close(out)
	}()
	return out
}

func checkImport(in <-chan interface{}, reg *regexp.Regexp) <-chan interface{} {
	out := make(chan interface{})
	counts := []int{}
	go func() {
		for msg := range in {
			t := msg.(types.ContentMsg)
			if reg.MatchString(t.FileContent) {
				if CheckFalsePositives {
					// checks imports of Java collection-like libs
					for i, inspectedLib := range InspectedLibs {
						if inspectedLib.Regex.MatchString(t.FileContent) {
							fmt.Println(t.InnerFileName)
							counts[i]++
						}
					}
				}
				out <- t
			}
		}
		if CheckFalsePositives {
			for i, inspectedLib := range InspectedLibs {
				fmt.Printf("%s File Count: %d\n", inspectedLib.Name, counts[i])
			}
		}
		close(out)
	}()
	return out
}

// broadcast to operator counters(workers)
func dispatchToOpsCounters(in <-chan interface{}, inOps []chan types.ContentMsg) {
	go func() {
		for msg := range in {
			t := msg.(types.ContentMsg)
			for _, inOp := range inOps {
				inOp <- t
			}
		}
		// closes all channels when done
		types.CloseAllInOps(inOps)
	}()
}

func gatherResults(outOps <-chan interface{}, result *orderedmap.OrderedMap) <-chan int {
	out := make(chan int)
	go func() {
		countFiles := 0
		for msg := range outOps {
			countMsg := msg.(types.CountMsg)
			v, _ := result.Get(countMsg.FileName)
			mapEntry := v.(*orderedmap.OrderedMap)
			v, _ = mapEntry.Get(countMsg.OperatorCount.Operator)
			count := v.(int)
			mapEntry.Set(countMsg.OperatorCount.Operator,
				count+countMsg.OperatorCount.Total)
			countFiles++
			if CheckFalsePositives {
				// accumulates total of operators occurrences in files w/ collection-like libs' imports
				// used for checking of Java collection-like libs
				// extremely usabled for sample inspection
				if elem, ok := FilesMap[countMsg.InnerFileName]; ok {
					if countMsg.OperatorCount.Total > 0 {
						elem.Set(countMsg.OperatorCount.Operator, countMsg.OperatorCount.Total)
					}
				}
			}
		}

		if CheckFalsePositives {
			// saves ops' count from files w/ collection-like libs' imports in a pretty JSON file
			// used to facilitate the looking for false positives in those files
			util.WritePrettyJSON(filepath.Join(config.FALSE_POSITIVES_PATH, "collection-like_count"),
				FilesMap)
		}

		// sort results
		result.SortKeys(sort.Strings)
		// sort each entry by operators' name
		for _, k := range result.Keys() {
			v, _ := result.Get(k)
			entry := v.(*orderedmap.OrderedMap)
			entry.SortKeys(sort.Strings)
		}
		out <- countFiles
		close(out)
	}()
	return out
}
