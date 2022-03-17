package util

import (
	"encoding/json"
	"log"
	"os"
	"strings"

	"github.com/golang-module/carbon/v2"
)

func CheckError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func WriteJSON(path string, data interface{}) {
	j, err := json.Marshal(data)
	CheckError(err)

	err = os.WriteFile(path+".json", j, 0644)
	CheckError(err)
}

func WritePrettyJSON(path string, data interface{}) {
	j, err := json.MarshalIndent(data, "", "\t")
	CheckError(err)

	err = os.WriteFile(path+".json", j, 0644)
	CheckError(err)
}

func WriteFolder(folderPath string) {
	err := os.MkdirAll(folderPath, os.ModePerm)
	CheckError(err)
}

func RemoveAllFolders(folderPath string) {
	err := os.RemoveAll(folderPath)
	CheckError(err)
}

func NowDateTimeFormatted() string {
	return strings.ReplaceAll(carbon.Now().ToDateTimeString(), ":", "-")
}
