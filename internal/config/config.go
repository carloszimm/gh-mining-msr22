package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"github.com/carloszimm/github-mining/internal/util"
)

var (
	REPO_RETRIVAL_PATH    = filepath.Join("assets", "repo-retrieval")
	OPERATORS_PATH        = filepath.Join("assets", "operators")
	OPERATORS_SEARCH_PATH = filepath.Join("assets", "operators-search")
)
var PROCESSING_WORKERS = runtime.NumCPU()

const ARCHIVES_FOLDER = "archives"

type Config struct {
	Tokens         []string `json:"tokens" validate:"required"`
	Distribution   string   `json:"distribution" validate:"required"`
	MinStars       int      `json:"min_stars" validate:"required"`
	IncreaseFactor int      `json:"increase_factor" validate:"required"`
}

var instance *Config

func init() {
	instance = readConfig()
}

func readConfig() *Config {
	dat, err := os.ReadFile(filepath.Join("configs", "config.json"))
	util.CheckError(err)

	var config Config
	json.Unmarshal(dat, &config)

	return &config
}

func GetConfigInstance() *Config {
	return instance
}
