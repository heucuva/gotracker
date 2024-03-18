package config

import (
	"errors"
	"os"

	"github.com/hashicorp/go-envparse"
)

type envfile struct {
	environment
}

var EnvFile Overlay = envfile{}

func (envfile) Wants(param any) bool {
	return true
}

func (o envfile) Update(cfg, _ any) error {
	return o.loadEnv(cfg, ".env")
}

func (o envfile) loadEnv(cfg any, filename string) error {
	f, err := os.Open(filename)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if f == nil {
		return nil
	}

	defer f.Close()

	m, err := envparse.Parse(f)
	if err != nil {
		return err
	}

	return o.environment.updateFrom(cfg, func(s string) (string, bool) {
		v, found := m[s]
		return v, found
	})
}
