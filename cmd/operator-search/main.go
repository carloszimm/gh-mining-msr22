package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/carloszimm/github-mining/internal/config"
	"github.com/carloszimm/github-mining/internal/processing"
	"github.com/carloszimm/github-mining/internal/types"
	"github.com/carloszimm/github-mining/internal/util"
	"github.com/iancoleman/orderedmap"
)

type LangExtension struct {
	Name         string   `json:"name"`
	TypeLanguage string   `json:"-"`
	Extensions   []string `json:"extensions"`
}

var languageExtensions []LangExtension

func init() {
	// load the languages' extensions
	dat, err := os.ReadFile(filepath.Join("assets", "Programming_Languages_Extensions.json"))
	util.CheckError(err)

	err = json.Unmarshal(dat, &languageExtensions)
	util.CheckError(err)
}

func loadExtensions(cfg *config.Config) map[string]struct{} {
	extensions := make(map[string]struct{})
	for _, fileExt := range cfg.FileExtensions {
		for _, exts := range languageExtensions {
			if fileExt == exts.Name {
				for _, ext := range exts.Extensions {
					extensions[ext] = struct{}{}
				}
			}
		}
	}
	return extensions
}

func loadOperators(cfg *config.Config) *types.Operators {
	opDir, err := os.ReadDir(config.OPERATORS_PATH)
	util.CheckError(err)

	var operators *types.Operators
	// (?i) case insensitive
	reg := regexp.MustCompile("(?i)" + cfg.Distribution)
	for _, d := range opDir {
		if !d.IsDir() && reg.MatchString(d.Name()) {
			operators = types.CreateOperators(d.Name(), cfg.Distribution)
			break
		}
	}

	return operators
}

func createResultMap(cfg *config.Config, operatorsList []string) *orderedmap.OrderedMap {
	// loads info about the files in archives(repositories)
	dat, err := os.ReadFile(filepath.Join(config.REPO_RETRIVAL_PATH, cfg.Distribution, "list_of_files.json"))
	util.CheckError(err)

	var archivesInfos []types.InfoFile
	err = json.Unmarshal(dat, &archivesInfos)
	util.CheckError(err)

	result := orderedmap.New()
	// initializes result
	for _, val := range archivesInfos {
		if cfg.Distribution == "RxJS" && val.FileName == "zwacky-game-music-player-v1-38-g3171b55.tar.gz" {
			continue
		}
		result.Set(val.FileName, orderedmap.New())
		v, _ := result.Get(val.FileName)
		entry := v.(*orderedmap.OrderedMap)
		for _, op := range operatorsList {
			entry.Set(op, 0)
		}
	}

	return result
}

func main() {
	cfg := config.GetConfigInstance()
	log.Printf("Starting searching for %s operators", cfg.Distribution)

	flag.BoolVar(&processing.CheckJavaCollectionLike, "checkcollection", false,
		"indicates if the process should look for imports of Java collection-like libs")

	// loads the extensions related to the analyzed distribution
	extensions := loadExtensions(cfg)

	// loads operators
	operators := loadOperators(cfg)

	// initializes result
	result := createResultMap(cfg, operators.GetOperators())

	resultChannel := processing.SetupOpsPipeline(extensions, operators, result)

	countFiles := <-resultChannel

	log.Println("Search for operators finished!")
	log.Printf("Number of processed files: %d. Writing Results...\n", countFiles/len(operators.GetOperators()))
	util.WriteFolder(config.OPERATORS_SEARCH_PATH)
	fileName := fmt.Sprintf("%s_%s", strings.ToLower(cfg.Distribution), strings.Join(cfg.FileExtensions, "-"))
	util.WriteJSON(
		filepath.Join(config.OPERATORS_SEARCH_PATH, fileName),
		result)
	log.Println("Done!")
}
