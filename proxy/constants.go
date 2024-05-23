package proxy

import (
	"encoding/hex"
	"fmt"
	"runtime/debug"
	"time"
)

const (
	Version     = "0.1.0"
	ServiceName = "eth-proxy"
)

var (
	FullVersion = fmt.Sprintf("%s-%v", Version, gitCommitHash[0:8]) // FullVersion prints semantic version followed by commit hash

	//
	// https://icinga.com/blog/2022/05/25/embedding-git-commit-information-in-go-binaries/
	//
	gitCommit string // overwritten by -ldflag "-X 'github.com/ATMackay/eth-proxy/service.gitCommit=$commit_hash'"
	buildDate string // overwritten by -ldflag "-X 'github.com/ATMackay/eth-proxy/service.buildDate=$build_date'"
)

// gitCommitHash returns a string builder that reads information embedded
// in the running binary during the build process.
var gitCommitHash = func() string { return makeVCS() }()

func makeVCS() string {
	// Try embedded value
	if len(gitCommit) > 7 {
		mustDecodeHex(gitCommit[0:8]) // will panic if build has been generated with a malicious $commit_hash value
		return gitCommit[0:8]
	}
	var commit string
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				commit = setting.Value
			}
		}
	}
	if commit == "" {
		commit = "00000000" // default commit string
	}
	mustDecodeHex(commit)
	return commit
}

// date returns a formatted time.Time to string generator
var date = func() string { return makeDate() }()

func makeDate() string {
	if buildDate != "" {
		return buildDate
	}
	return time.Now().Format(time.RFC3339)
}

func mustDecodeHex(input string) {
	_, err := hex.DecodeString(input)
	if err != nil {
		panic(err)
	}
}
