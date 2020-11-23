package zap

import (
	"io/ioutil"
	"os"
	"strings"

	"github.com/puma/puma-dev/homedir"
	"gopkg.in/yaml.v2"
)

const appsPath = "~/.zap"

// AppConfig holds the configuration for a given host host
type AppConfig struct {
	Scheme  string
	Host    string
	Path    string
	Dir     string `json:",omitempty"`
	Content string `json:",omitempty"`
	Key     string
}

func getAppConfig(host string) (*AppConfig, error) {
	path, stat, err := getClosestMatchingPath(host)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		dir, err := os.Readlink(path)
		if err != nil {
			return nil, err
		}

		return &AppConfig{
			Host: host,
			Path: path,
			Dir:  dir,
			Key:  dir,
		}, nil
	}

	config, err := readConfigFromFile(path, host)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func readConfigFromFile(path, host string) (*AppConfig, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &AppConfig{Scheme: "http"}
	err = yaml.Unmarshal([]byte(data), config)
	if err != nil {
		return nil, err
	}

	config.Host = host
	config.Key = config.Dir
	if config.Key == "" {
		// FIXME: host is coded in the proxy so we need one per host source
		config.Key = host + "->" + config.Content
	}

	return config, nil
}

// recursively finds the closest matching config path for a given host
// used to match subdomains automatically, eg. if you request moo.foo.dev
// it will check moo.foo.dev and foo.dev in that order and return the first
// one it finds
func getClosestMatchingPath(host string) (string, os.FileInfo, error) {
	path := homedir.MustExpand(appsPath) + "/" + host
	stat, err := os.Stat(path)
	if err != nil {
		parts := strings.Split(host, ".")
		if len(parts) > 2 {
			return getClosestMatchingPath(strings.Join(parts[1:], "."))
		}
		return path, nil, err
	}
	return path, stat, nil
}
