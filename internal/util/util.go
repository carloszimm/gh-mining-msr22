package util

import (
	"encoding/json"
	"log"
	"os"
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
