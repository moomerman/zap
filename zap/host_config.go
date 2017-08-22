package zap

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"strconv"

	"github.com/puma/puma-dev/homedir"
)

// HostConfig holds the configuration for a given host host
type HostConfig struct {
	Host    string
	Path    string
	Dir     string
	Content string
	Key     string
}

func getHostConfig(host string) (*HostConfig, error) {
	path := homedir.MustExpand(appsPath) + "/" + host
	stat, err := os.Stat(path)
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
