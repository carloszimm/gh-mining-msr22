package processing

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/carloszimm/github-mining/internal/config"
	"github.com/iancoleman/orderedmap"
)

type InspectedLib struct {
	Name  string
	Regex *regexp.Regexp
}

var InspectedLibs = []InspectedLib{
	{Name: "Stream", Regex: regexp.MustCompile(`java\.util\.stream`)},
	{Name: "Eclipse", Regex: regexp.MustCompile(`org\.eclipse\.collections`)},
	{Name: "CollectionUtils", Regex: regexp.MustCompile(`org\.apache\.commons\.collections4\.CollectionUtils`)},
	{Name: "Guava", Regex: regexp.MustCompile(`com\.google\.common\.collect\.Collections2|com\.google\.guava`)},
}

//const FILE = "collection-like_files.txt"
const FILE = "collection-like_sample.txt"

var FilesMap map[string]*orderedmap.OrderedMap

func init() {
	FilesMap = make(map[string]*orderedmap.OrderedMap)
	file, err := os.Open(filepath.Join(config.FALSE_POSITIVES_PATH, FILE))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// goes through line by line
	for scanner.Scan() {
		fileName := strings.TrimSpace(scanner.Text())
		if _, ok := FilesMap[fileName]; !ok {
			FilesMap[fileName] = orderedmap.New()
		}
	}
	fmt.Println("Total of Collection-like files:", len(FilesMap))

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
