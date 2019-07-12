package oneliners

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/hokaccha/go-prettyjson"
	"log"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
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

func PrettyJson(a interface{}, msg ...string) {
	_, file, ln, ok := runtime.Caller(1)
	if ok {
		fmt.Println("__FILE__", file, "__LINE__", ln)
		if a != nil {
			// ref: https://stackoverflow.com/questions/37770005/why-is-there-no-byte-kind-in-the-reflect-package
			if reflect.TypeOf(a).String() == "[]uint8" {
				var js interface{}
				if err := json.Unmarshal(a.([]byte), &js); err == nil {
					a = js
				}
			}
			f := prettyjson.NewFormatter()
			f.KeyColor = color.New(color.FgHiMagenta)
			f.StringColor = color.New(color.FgGreen)
			f.BoolColor = color.New(color.FgHiYellow)
			f.NumberColor = color.New(color.FgHiCyan)
			f.NullColor = color.New(color.FgHiBlue, color.Italic)
			data, _ := f.Marshal(a)
			str := ""
			if len(msg) > 0 {
				str = strings.Trim(fmt.Sprintf("%v", msg), "[]")
				str = fmt.Sprintf("[ %s ]", str)
			}
			fmt.Printf("============================%s============================\n", str)
			fmt.Println(string(data))
		}
	} else {
		log.Fatal("Failed to detect runtime caller info.")
	}
}

// Deprecated use debug.PrintStack() directly
func PrintStacktrace() {
	debug.PrintStack()
}
