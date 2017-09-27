package zap

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/puma/puma-dev/homedir"
)

const appsPath = "~/.zap"

// HostConfig holds the configuration for a given host host
type HostConfig struct {
	Host    string
	Path    string
	Dir     string
	Content string
	Key     string
}

func getHostConfig(host string) (*HostConfig, error) {
	path, stat, err := getClosestMatchingPath(host)
	if err != nil {
		return nil, err
	}

	if stat.IsDir() {
		dir, err := os.Readlink(path)
		if err != nil {
			return nil, err
		}

		return &HostConfig{
			Host: host,
			Path: path,
			Dir:  dir,
			Key:  dir,
		}, nil
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	data = bytes.TrimSpace(data)

	var proxy string

	port, err := strconv.Atoi(string(data))
	if err == nil {
		proxy = "http://127.0.0.1:" + strconv.Itoa(port)
	} else {
		u, err := url.Parse(string(data))
		if err != nil {
			return nil, err
		}

		host, sport, err := net.SplitHostPort(u.Host)
		if err == nil {
			port, err = strconv.Atoi(sport)
			if err != nil {
				return nil, err
			}
			proxy = u.Scheme + "://" + host + ":" + strconv.Itoa(port)
		} else {
			host = u.Host
			proxy = u.Scheme + "://" + host
		}

	}

	return &HostConfig{
		Host:    host,
		Path:    path,
		Content: proxy,
		Key:     host + "->" + proxy, // FIXME: host is coded in the proxy so we need one per host source
	}, nil
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
