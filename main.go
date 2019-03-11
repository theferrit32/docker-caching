package main

import (
	"fmt"
	"os"

	"github.com/juju/loggo"
)

var logger = loggo.GetLogger("main")

func ensureFileGone(path string) {
	err := os.Remove(path)
	if err != nil {
		logger.Warningf(err.Error())
	}
}

func end() {
	ensureFileGone(sockPath)
	os.Exit(1)
}

func main() {
	fmt.Printf("hello\n")
	sockPath := "/tmp/testsock.sock"
	destSockPath := "/run/docker.sock"
	ensureFileGone(sockPath)
	unix_domainsocket_proxy(sockPath, destSockPath)
}
