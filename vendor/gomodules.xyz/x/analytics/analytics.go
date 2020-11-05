package analytics

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"gomodules.xyz/homedir"
)

func ClientID() string {
	dir := filepath.Join(homedir.HomeDir(), ".appscode")
	filename := filepath.Join(dir, "client-id")
	id, err := ioutil.ReadFile(filename)
	if os.IsNotExist(err) {
		id := uuid.New().String()
		if e2 := os.MkdirAll(dir, 0755); e2 == nil {
			ioutil.WriteFile(filename, []byte(id), 0644)
		}
		return id
	}
	return string(id)
}
