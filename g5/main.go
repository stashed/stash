package main

import (
	"github.com/codeskyblue/go-sh"
	"fmt"
)

func main() {
	sh.Command("echo", "hello").Run()

	session := sh.NewSession()
	session.SetEnv("BUILD_ID", "123")
	session.SetDir("/")
	session.Command("echo", "hello").Run()
	session.ShowCMD = true

	o, _ := session.Command("env").Output()
	fmt.Println(string(o))

	session.SetEnv("BUILD_ID", "12356")
	o2, _ := session.Command("env").Output()
	fmt.Println(string(o2))
}
