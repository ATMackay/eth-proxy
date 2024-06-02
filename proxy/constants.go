package proxy

import (
	"time"
)

const (
	ServiceName = "eth-proxy"
)

var (
	Version   = "0.1.0" // overwritten by -ldflag "-X 'github.com/ATMackay/eth-proxy/proxy.Version=$version_tag'"
	GitCommit = ""      // overwritten by -ldflag "-X 'github.com/ATMackay/eth-proxy/proxy.GitCommit=$commit_hash'"

	CommitDate = "" // overwritten by -ldflag "-X 'github.com/ATMackay/eth-proxy/proxy.CommitDate=$build_date'"
	BuildDate  = time.Now().Format(time.DateTime)
)
