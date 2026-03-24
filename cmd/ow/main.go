package main

import (
	"fmt"
	"runtime"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	fmt.Printf("ow %s %s %s %s/%s\n", version, commit, date, runtime.GOOS, runtime.GOARCH)
}
