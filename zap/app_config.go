package zap

import (
	"os"
	"strings"

	"github.com/moomerman/zap/internal/homedir"
	"go.yaml.in/yaml/v3"
)

const appsPath = "~/.zap"

// AppConfig holds the configuration for a given host host
type AppConfig struct {
	Scheme  string
	Host    string
	Port    string
	Path    string
	Dir     string `json:",omitempty"`
	Command string `json:",omitempty"`
	Proxy   string `json:",omitempty"`
	Key     string
}

func getAppConfig(host string) (*AppConfig, error) {
	path, err := getClosestMatchingPath(host)
	if err != nil {
		return nil, err
	}

	config, err := readConfigFromFile(path, host)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func readConfigFromFile(path, host string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &AppConfig{Scheme: "http", Port: "PORT"}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	config.Host = host
	config.Key = config.Dir
	if config.Key == "" {
		// FIXME: host is coded in the proxy so we need one per host source
		config.Key = host + "->" + config.Proxy
	}

	return config, nil
}

// recursively finds the closest matching config path for a given host
// used to match subdomains automatically, eg. if you request moo.foo.dev
// it will check moo.foo.dev and foo.dev in that order and return the first
// one it finds
func getClosestMatchingPath(host string) (string, error) {
	dir, err := homedir.Expand(appsPath)
	if err != nil {
		return "", err
	}
	path := dir + "/" + host
	_, err = os.Stat(path)
	if err != nil {
		parts := strings.Split(host, ".")
		if len(parts) > 2 {
			return getClosestMatchingPath(strings.Join(parts[1:], "."))
		}
		return path, err
	}
	return path, nil
}
