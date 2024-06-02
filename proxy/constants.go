package proxy

import (
	"time"
)

const (
	ServiceName = "eth-proxy"
)

var (
	versionTag = "0.1.0" // overwritten by -ldflag "-X 'github.com/ATMackay/eth-proxy/proxy.versionTag=$version_tag'"
	gitCommit  = ""      // overwritten by -ldflag "-X 'github.com/ATMackay/eth-proxy/proxy.gitCommit=$commit_hash'"

	Version = versionTag

	CommitDate = "" // overwritten by -ldflag "-X 'github.com/ATMackay/eth-proxy/proxy.CommitDate=$build_date'"
	BuildDate  = time.Now().Format(time.DateTime)
)
