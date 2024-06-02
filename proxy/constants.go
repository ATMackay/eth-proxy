package proxy

import (
	"encoding/hex"
	"fmt"
	"runtime/debug"
	"time"
)

const (
	ServiceName = "eth-proxy"
)

var (
	versionTag = "0.1.0" // overwritten by -ldflag "-X 'github.com/ATMackay/eth-proxy/proxy.versionTag=$version_tag'"
	gitCommit  = ""      // overwritten by -ldflag "-X 'github.com/ATMackay/eth-proxy/proxy.gitCommit=$commit_hash'"

	Version = fmt.Sprintf("%v-%v", versionTag, gitCommitHash[0:8])

	CommitDate = "" // overwritten by -ldflag "-X 'github.com/ATMackay/eth-proxy/proxy.CommitDate=$build_date'"
	BuildDate  = time.Now().Format(time.DateTime)
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
func mustDecodeHex(input string) {
	_, err := hex.DecodeString(input)
	if err != nil {
		panic(err)
	}
}
