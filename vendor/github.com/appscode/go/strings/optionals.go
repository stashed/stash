package strings

import "log"

func VString(def string, args ...string) string {
	v := def
	if len(args) == 1 {
		v = args[0]
	} else if len(args) > 1 {
		v = args[0]
		log.Printf("Found more than 1 argument when expected 1 %v", args)
	}
	return v
}
