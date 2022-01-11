package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/carloszimm/github-mining/internal/config"
	"github.com/carloszimm/github-mining/internal/processing"
	"github.com/carloszimm/github-mining/internal/types"
	"github.com/carloszimm/github-mining/internal/util"
	"github.com/golang-module/carbon/v2"
	"github.com/iancoleman/orderedmap"
)

var ALLOWED_EXTENSIONS = map[string][]string{
	"RxJava":  {"Java"},
	"RxJS":    {"JavaScript", "CoffeeScript", "TypeScript", "JSX"},
	"RxSwift": {"Swift"},
}

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

func main() {
	cfg := config.GetConfigInstance()
	log.Printf("Starting searching for %s at %s\n", cfg.Distribution, carbon.Now().ToDayDateTimeString())

	allowedExtensions := ALLOWED_EXTENSIONS[cfg.Distribution]

	// loads the extensions related to the analyzed distribution
	extensions := make(map[string]struct{})
	for _, allowedExt := range allowedExtensions {
		for _, exts := range languageExtensions {
			if allowedExt == exts.Name {
				for _, ext := range exts.Extensions {
					extensions[ext] = struct{}{}
				}
			}
		}
	}
	// loads operators
	opDir, err := os.ReadDir(config.OPERATORS_PATH)
	util.CheckError(err)

	var operators *types.Operators
	for _, d := range opDir {
		if !d.IsDir() && strings.Contains(strings.ToLower(d.Name()), strings.ToLower(cfg.Distribution)) {
			operators = types.CreateOperators(d.Name(), cfg.Distribution)
		}
	}

	// loads info about the files in archives(repositories)
	dat, err := os.ReadFile(filepath.Join(config.REPO_RETRIVAL_PATH, cfg.Distribution, "list_of_files.json"))
	util.CheckError(err)

	var archivesInfos []types.InfoFile
	err = json.Unmarshal(dat, &archivesInfos)
	util.CheckError(err)

	result := orderedmap.New()
	// initializes result
	for _, val := range archivesInfos {
		result.Set(val.FileName, orderedmap.New())
		for _, op := range operators.GetOperators() {
			v, _ := result.Get(val.FileName)
			entry := v.(*orderedmap.OrderedMap)
			entry.Set(op, 0)
		}
	}

	resultChannel := processing.SetupOpsPipeline(extensions, operators, result)

	result = <-resultChannel

	log.Printf("Search for operators finished at: %s\n", carbon.Now().ToDayDateTimeString())
	log.Println("Writing Results...")
	util.WriteFolder(config.OPERATORS_SEARCH_PATH)
	util.WriteJSON(filepath.Join(config.OPERATORS_SEARCH_PATH, strings.ToLower(cfg.Distribution)), result)
	log.Println("Done!")
}
