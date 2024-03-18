package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type pflags struct{}

var PFlags Overlay = pflags{}

func (pflags) Wants(param any) bool {
	_, ok := param.(*cobra.Command)
	return ok
}

func (pflags) Update(cfg, param any) error {
	cmd, ok := param.(*cobra.Command)
	if !ok {
		panic("params expected to be a *cobra.Command")
	}

	for info, f := range getFields(cfg) {
		var name, shorthand string
		var fs *pflag.FlagSet
		if n, ok := info.Tag.Lookup("pflag"); ok && len(n) > 0 {
			name = n
			fs = cmd.PersistentFlags()

			if v, ok := info.Tag.Lookup("pf"); ok {
				shorthand = v
			}
		} else if n, ok := info.Tag.Lookup("flag"); ok && len(n) > 0 {
			name = n
			fs = cmd.Flags()

			if v, ok := info.Tag.Lookup("f"); ok {
				shorthand = v
			}
		} else {
			continue
		}

		var usage string
		if v, ok := info.Tag.Lookup("usage"); ok {
			usage = v
		}

		var err error
		switch v := f.Interface().(type) {
		case *bool:
			fs.BoolVarP(v, name, shorthand, *v, usage)
		case *int64:
			fs.Int64VarP(v, name, shorthand, *v, usage)
		case *int32:
			fs.Int32VarP(v, name, shorthand, *v, usage)
		case *int16:
			fs.Int16VarP(v, name, shorthand, *v, usage)
		case *int8:
			fs.Int8VarP(v, name, shorthand, *v, usage)
		case *int:
			fs.IntVarP(v, name, shorthand, *v, usage)
		case *uint64:
			fs.Uint64VarP(v, name, shorthand, *v, usage)
		case *uint32:
			fs.Uint32VarP(v, name, shorthand, *v, usage)
		case *uint16:
			fs.Uint16VarP(v, name, shorthand, *v, usage)
		case *uint8:
			fs.Uint8VarP(v, name, shorthand, *v, usage)
		case *uint:
			fs.UintVarP(v, name, shorthand, *v, usage)
		case *string:
			fs.StringVarP(v, name, shorthand, *v, usage)
		case *[]bool:
			fs.BoolSliceVarP(v, name, shorthand, *v, usage)
		case *[]int64:
			fs.Int64SliceVarP(v, name, shorthand, *v, usage)
		case *[]int32:
			fs.Int32SliceVarP(v, name, shorthand, *v, usage)
		case *[]int:
			fs.IntSliceVarP(v, name, shorthand, *v, usage)
		case *[]string:
			fs.StringSliceVarP(v, name, shorthand, *v, usage)
		default:
			err = fmt.Errorf("unhandled type: %T", f.Interface())
		}

		if err != nil {
			return fmt.Errorf("error while parsing %q: %w", name, err)
		}
	}

	return nil
}
