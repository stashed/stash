package oneliners

import (
	"fmt"
	"net/http"
	"net/http/httputil"
)

func DumpHttpRequest(req *http.Request) {
	fmt.Println()
	fmt.Println("REQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ----------------------------------------------------")
	b, err := httputil.DumpRequest(req, true)
	if err == nil {
		fmt.Println(string(b))
	} else {
		fmt.Println(err)
	}
	fmt.Println("----------------------------------------------------REQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ")
	fmt.Println()
}

func DumpHttpRequestOut(req *http.Request) {
	fmt.Println()
	fmt.Println("REQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ----------------------------------------------------")
	b, err := httputil.DumpRequestOut(req, true)
	if err == nil {
		fmt.Println(string(b))
	} else {
		fmt.Println(err)
	}
	fmt.Println("----------------------------------------------------REQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQQ")
	fmt.Println()
}

func DumpHttpResponse(resp *http.Response) {
	fmt.Println()
	fmt.Println("RESPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPP----------------------------------------------------")
	b, err := httputil.DumpResponse(resp, true)
	if err == nil {
		fmt.Println(string(b))
	} else {
		fmt.Println(err)
	}
	fmt.Println("----------------------------------------------------RESPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPPP")
	fmt.Println()
}
