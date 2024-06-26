package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/ATMackay/eth-proxy/proxy"
	"github.com/vrischmann/envconfig"
	yaml "gopkg.in/yaml.v3"
)

const envPrefix = "ETH_PROXY"

var (
	configFilePath string
	configFilePtr  = flag.String("config", "config.yml", "path to config file")
)

// RUN WITH PLAINTEXT CONFIG [RECOMMENDED FOR TESTING ONLY]
// $ go run main.go --config ./config.yml
// $ go run main.go --config {path_to_config_file}
//
// OR RUN WITH ENVIRONMENT VARIABLES
//
// $ go build
// $ export ETH_PROXY_URLS=<client_url>
// $ ./eth-proxy
//
//

func init() {
	// Parse flag containing path to config file
	flag.Parse()
	if configFilePtr != nil {
		configFilePath = *configFilePtr
	}
}

// parseYAMLConfig parse configuration file or environment variables, receiver must be a pointer
func parseYAMLConfig(configFile string, receiver any, prefix string) error {
	b, err := os.ReadFile(filepath.Clean(configFile))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if b != nil {
		if err := yaml.Unmarshal(b, receiver); err != nil {
			return err
		}
	}
	// environment variables supersede config yaml files
	if err := envconfig.InitWithOptions(receiver, envconfig.Options{Prefix: prefix, AllOptional: true}); err != nil {
		return err
	}
	return nil
}

func main() {

	var cfg proxy.Config

	if err := parseYAMLConfig(configFilePath, &cfg, envPrefix); err != nil {
		panic(fmt.Sprintf("error parsing config: %v", err))
	}

	cfg.Sanitize()

	l, err := proxy.NewLogger(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		panic(err)
	}

	multiClient, err := proxy.NewMultiNodeClient(cfg.URLs, proxy.NewEthClient)
	if err != nil {
		panic(err)
	}

	srv := proxy.New(cfg.Port, l, multiClient)

	srv.Start()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	sig := <-sigChan
	srv.Stop(sig)
}
