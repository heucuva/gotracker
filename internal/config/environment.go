package config

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

type environment struct{}

var Environment Overlay = environment{}

func (environment) Wants(_ any) bool {
	return true
}

func (o environment) Update(cfg, _ any) error {
	return o.updateFrom(cfg, os.LookupEnv)
}

func (environment) updateFrom(cfg any, lookup func(string) (string, bool)) error {
	for info, f := range getFields(cfg) {
		name, ok := info.Tag.Lookup("env")
		if !ok || len(name) == 0 || name == "-" {
			continue
		}

		envName := strings.ToUpper(strings.Join(strings.FieldsFunc(name, func(r rune) bool {
			return !(unicode.IsLetter(r) || unicode.IsDigit(r))
		}), "_"))

		val, found := lookup(envName)
		if !found {
			continue
		}

		if err := setValueToField(f, val); err != nil {
			return fmt.Errorf("error while parsing %q (%s): %w", name, envName, err)
		}
	}

	return nil
}
