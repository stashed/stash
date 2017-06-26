package oneliners

import (
	"fmt"
	"log"
	"runtime"
)

func FILE(a ...interface{}) {
	_, file, ln, ok := runtime.Caller(1)
	if ok {
		fmt.Println("__FILE__", file, "__LINE__", ln)
		if len(a) > 0 {
			fmt.Println(a...)
		}
	} else {
		log.Fatal("Failed to detect runtime caller info.")
	}
}
