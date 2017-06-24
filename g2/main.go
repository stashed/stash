package main

import (
	"path/filepath"
	"fmt"
)

func main() {
	fmt.Println(filepath.Join("a", ""))
	fmt.Println(filepath.Join("b", "c"))
}
