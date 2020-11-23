package zap

import (
	"log"
	"os"
	"path"

	"github.com/moomerman/zap/adapter"
	"github.com/moomerman/zap/adapter/buffalo"
	"github.com/moomerman/zap/adapter/hugo"
	"github.com/moomerman/zap/adapter/phoenix"
	"github.com/moomerman/zap/adapter/rails"
	"github.com/moomerman/zap/adapter/static"
)

// GetAdapter returns the corresponding adapter for the given
// host/dir combination
func GetAdapter(scheme, host, dir string) (adapter.Adapter, error) {
	_, err := os.Stat(path.Join(dir, "mix.exs"))
	if err == nil {
		log.Println("[app]", host, "using the phoenix adapter (found mix.exs)")
		return phoenix.New(scheme, host, dir), nil
	}

	_, err = os.Stat(path.Join(dir, "Gemfile"))
	if err == nil {
		log.Println("[app]", host, "using the rails adapter (found Gemfile)")
		return rails.New(scheme, host, dir), nil
	}

	_, err = os.Stat(path.Join(dir, ".buffalo.dev.yml"))
	if err == nil {
		log.Println("[app]", host, "using the buffalo adapter (found .buffalo.dev.yml)")
		return buffalo.New(scheme, host, dir), nil
	}

	_, err = os.Stat(path.Join(dir, "config.toml"))
	if err == nil {
		log.Println("[app]", host, "using the hugo adapter (found config.toml)")
		return hugo.New(scheme, host, dir), nil
	}

	log.Println("[app]", host, "using the static adapter")
	return static.New(dir)
}
