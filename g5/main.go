package main

import (
	"path/filepath"
	"fmt"
)

func main() {
 fmt.Printf(filepath.Join("a", "", "b"))
}
