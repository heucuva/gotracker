package command

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	libListFieldName     bool = true
	libListFieldHomepage bool = true
	libListFieldPackage  bool = false
	libListFieldLicense  bool = true
)

type libraryInfo struct {
	Name     string
	Package  string
	Homepage string
	License  string
}

var librariesList = []libraryInfo{
	{
		Name:     "x/sys",
		Package:  "golang.org/x/sys",
		License:  "BSD-3-Clause",
		Homepage: "",
	},
	{
		Name:     "yaml",
		Package:  "gopkg.in/yaml.v2",
		License:  "Apache-2.0",
		Homepage: "https://github.com/go-yaml/yaml",
	},
	{
		Name:     "go-md2man",
		Package:  "github.com/cpuguy83/go-md2man/v2/md2man",
		License:  "MIT",
		Homepage: "https://github.com/cpuguy83/go-md2man",
	},
	{
		Name:     "sanitized_anchor_name",
		Package:  "github.com/shurcooL/sanitized_anchor_name",
		License:  "MIT",
		Homepage: "https://github.com/shurcooL/sanitized_anchor_name",
	},
	{
		Name:     "Blackfriday",
		Package:  "github.com/russross/blackfriday/v2",
		License:  "BSD-2-Clause",
		Homepage: "https://github.com/russross/blackfriday",
	},
	{
		Name:     "SemVer",
		Package:  "github.com/Masterminds/semver",
		License:  "MIT",
		Homepage: "https://github.com/Masterminds/semver",
	},
	{
		Name:     "mousetrap",
		Package:  "github.com/inconshreveable/mousetrap",
		License:  "Apache-2.0",
		Homepage: "https://github.com/inconshreveable/mousetrap",
	},
	{
		Name:     "pflag",
		Package:  "github.com/spf13/pflag",
		License:  "BSD-3-Clause",
		Homepage: "https://github.com/spf13/pflag",
	},
	{
		Name:     "OPL2",
		Package:  "github.com/gotracker/opl2",
		License:  "GPL-2.0",
		Homepage: "https://github.com/gotracker/opl2",
	},
	{
		Name:     "Terminal progress bar for Go",
		Package:  "github.com/cheggaaa/pb",
		License:  "BSD-3-Clause",
		Homepage: "https://github.com/cheggaaa/pb",
	},
	{
		Name:     "goaudiofile",
		Package:  "github.com/gotracker/goaudiofile",
		License:  "Unlicense",
		Homepage: "https://github.com/gotracker/goaudiofile",
	},
	{
		Name:     "github.com/gotracker/gotracker/playback",
		Package:  "github.com/gotracker/playback",
		License:  "Unlicense",
		Homepage: "https://github.com/gotracker/playback",
	},
	{
		Name:     "go-runewidth",
		Package:  "github.com/mattn/go-runewidth",
		License:  "MIT",
		Homepage: "https://github.com/mattn/go-runewidth",
	},
	{
		Name:     "go-winmm",
		Package:  "github.com/heucuva/go-winmm",
		License:  "Unlicense",
		Homepage: "https://github.com/heucuva/go-winmm",
	},
	{
		Name:     "Cobra",
		Package:  "github.com/spf13/cobra",
		License:  "Apache-2.0",
		Homepage: "https://github.com/spf13/cobra",
	},
}

func init() {
	if flags := librariesListCmd.Flags(); flags != nil {
		flags.BoolVarP(&libListFieldName, "name", "n", libListFieldName, "display library name")
		flags.BoolVarP(&libListFieldHomepage, "homepage", "H", libListFieldHomepage, "display library homepage URL")
		flags.BoolVarP(&libListFieldPackage, "package", "p", libListFieldPackage, "display library package")
		flags.BoolVarP(&libListFieldLicense, "license", "l", libListFieldLicense, "display library license type")
	}
	librariesCmd.AddCommand(librariesListCmd)
}

var (
	librariesListCmd = &cobra.Command{
		Use:   "list",
		Short: "List the libraries used in Gotracker",
		Long:  `List the libraries used in Gotracker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			nameIdx := make(map[string]int)
			var names []string

			for idx, lib := range librariesList {
				nameIdx[lib.Name] = idx
				names = append(names, lib.Name)
			}

			sort.Strings(names)

			var fields []string
			if libListFieldName {
				fields = append(fields, "Name")
			}
			if libListFieldPackage {
				fields = append(fields, "Package")
			}
			if libListFieldHomepage {
				fields = append(fields, "Homepage")
			}
			if libListFieldLicense {
				fields = append(fields, "License")
			}

			if len(fields) == 0 {
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
			fmt.Fprintln(tw, strings.Join(fields, "\t"))
			for _, name := range names {
				idx := nameIdx[name]
				lib := librariesList[idx]
				v := reflect.ValueOf(lib)

				for i, field := range fields {
					if i != 0 {
						fmt.Fprint(tw, "\t")
					}
					vf := v.FieldByName(field)
					fmt.Fprint(tw, vf.Interface())
				}
				fmt.Fprintln(tw)
			}
			return tw.Flush()
		},
	}
)
