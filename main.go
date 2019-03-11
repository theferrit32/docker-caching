package main

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/juju/loggo"
)

var logger = loggo.GetLogger("main")

func ensureFileGone(path string) {
	err := os.Remove(path)
	if err != nil {
		logger.Warningf(err.Error())
	}
}

func end(fileToDelete string) {
	ensureFileGone(fileToDelete)
	os.Exit(1)
}

func sliceEquals(sliceA []string, sliceB []string) bool {
	if len(sliceA) != len(sliceB) {
		return false
	}
	for i, a := range sliceA {
		if sliceB[i] != a {
			return false
		}
	}
	return true
}

/**
 * Takes fromImage value and overwrites it in the docker API request to point
 * to the registry passed in `redirectPullTo`
 */
func dockerPullRequestRedirectPreProcessor(redirectPullTo string) func(req *http.Request) bool {
	processor := func(req *http.Request) bool {
		fmt.Printf("processing Request for docker pull\n")
		changed := false
		dockerPullPathPattern :=
			"/v[\\d]+\\.[\\d]+/images/create\\?fromImage=[A-z0-9.-_]+&tag=[A-z0-9.-_]+"
		dockerPullUrlReg := regexp.MustCompile(dockerPullPathPattern)
		pathStr := req.URL.Path + "?" + req.URL.Query().Encode()
		matchedPull := dockerPullUrlReg.MatchString(pathStr)
		fmt.Printf("%s matched pattern?: %v\n", pathStr, matchedPull)
		if req.Method == "POST" && matchedPull {
			fmt.Println("was a docker pull")
			queryMap := req.URL.Query()
			fromImage := queryMap.Get("fromImage")
			fromImageTerms := strings.Split(fromImage, "/")
			if len(fromImageTerms) < 0 || len(fromImageTerms) > 2 {
				logger.Errorf("unknown fromImage specifier: %s\n", fromImage)
				return false
			} else if len(fromImageTerms) == 1 {
				logger.Debugf("fromImage didn't have a repo specified, setting to %s\n",
					redirectPullTo)
				fromImage = redirectPullTo + "/" + fromImage
			} else if fromImageTerms[0] != redirectPullTo {
				logger.Debugf("fromImage had repo %s specified, overwriting to %s\n",
					fromImageTerms[0], redirectPullTo)
				fromImage = redirectPullTo + "/" + fromImageTerms[1]
			} else {
				logger.Debugf("fromImage already matched desired value")
			}
			queryMap["fromImage"] = []string{fromImage}
			req.URL.RawQuery = queryMap.Encode()
		}
		return changed
	}
	return processor
}

func main() {
	logger.SetLogLevel(loggo.DEBUG)
	fmt.Printf("hello\n")
	sockPath := "/tmp/dockerproxy.sock"
	destSockPath := "/run/docker.sock"
	ensureFileGone(sockPath)

	redirectPullTo := "localhost:5000"
	unix_domain_socket_proxy(
		sockPath,
		destSockPath,
		dockerPullRequestRedirectPreProcessor(redirectPullTo))
}
