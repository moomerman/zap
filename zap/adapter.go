package zap

import (
	"log"

	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/adapter/server"
	"github.com/moomerman/zap/adapter/static"
)

// GetAdapter returns the corresponding adapter for the given config
func GetAdapter(config *AppConfig, onStatus adapter.StatusFunc, onLog func(string)) (adapter.Adapter, error) {
	if config.Command != "" {
		return server.New(&server.Config{
			Name:         "Server",
			Scheme:       config.Scheme,
			Host:         config.Host,
			Dir:          config.Dir,
			EnvPortName:  config.Port,
			ShellCommand: "exec " + config.Command + " # %s %s",
			OnStatus:     onStatus,
			OnLog:        onLog,
		}), nil
	}

	log.Println("[app]", config.Host, "using the static adapter")
	return static.New(config.Dir)
}
