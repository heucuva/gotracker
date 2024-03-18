package config

import (
	"os"
	"path/filepath"
)

type userenvfile struct {
	envfile
}

var UserEnvFile Overlay = userenvfile{}

func (userenvfile) Wants(param any) bool {
	return true
}

func (o userenvfile) Update(cfg, _ any) error {
	configPath, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return o.loadEnv(cfg, filepath.Join(configPath, ".env"))
}
