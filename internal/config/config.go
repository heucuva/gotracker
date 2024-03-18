package config

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/spf13/pflag"
)

var StandardOverlays = []Overlay{UserEnvFile, EnvFile, Environment, PFlags}

type Config[T any] struct {
	Values T
	order  []Overlay
}

func NewConfig[T any](d T) *Config[T] {
	return &Config[T]{
		Values: d,
	}
}

func (c *Config[T]) Overlay(overlays ...Overlay) *Config[T] {
	for _, o := range overlays {
		if !slices.Contains(c.order, o) {
			c.order = append(c.order, o)
		}
	}
	return c
}

func (c *Config[T]) If(predicate bool) *Config[T] {
	if predicate {
		return c
	}
	var dummy Config[T]
	return &dummy
}

func (c *Config[T]) Get() *T {
	return &c.Values
}

func (c *Config[T]) Update(param any) error {
	for _, o := range c.order {
		if !o.Wants(param) {
			continue
		}
		if err := o.Update(c, param); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config[T]) BoolVar(pf *pflag.FlagSet, name, fieldname, usage string) {
	f, err := getField[bool](c, fieldname)
	if err != nil {
		panic(err)
	}

	pf.BoolVar(f, name, *f, usage)
}

func (c *Config[T]) StringVar(pf *pflag.FlagSet, name, fieldname, usage string) {
	f, err := getField[string](c, fieldname)
	if err != nil {
		panic(err)
	}

	pf.StringVar(f, name, *f, usage)
}

func getField[TValue, T any](c *Config[T], fieldname string) (*TValue, error) {
	v := reflect.ValueOf(&c.Values).Elem()
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("config is not struct - got %T", c.Values)
	}

	var foundField bool
	fv := v.FieldByNameFunc(func(s string) bool {
		if s == fieldname {
			foundField = true
			return true
		}
		return false
	})
	if !foundField {
		return nil, fmt.Errorf("could not find field %q in struct", fieldname)
	}

	if !fv.CanInterface() {
		return nil, fmt.Errorf("could not interface field %q", fieldname)
	}

	var tv TValue
	if fv.Type() != reflect.TypeOf(tv) {
		return nil, fmt.Errorf("type of %q is not %T - got %T", fieldname, tv, fv.Interface())
	}

	addr := fv.Addr().Interface()

	return addr.(*TValue), nil
}

func getFields(cfg any) map[*reflect.StructField]reflect.Value {
	cv := reflect.ValueOf(cfg)
	vfv := cv.Elem().FieldByName("Values")
	if !vfv.IsValid() {
		return nil
	}

	m := map[*reflect.StructField]reflect.Value{}

	vfvt := vfv.Type()
	for i := 0; i < vfv.NumField(); i++ {
		if vt := vfvt.Field(i); vt.IsExported() {
			m[&vt] = vfv.FieldByIndex(vt.Index).Addr()
		}
	}

	return m
}
