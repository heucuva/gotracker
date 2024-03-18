package command

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/gotracker/gotracker/internal/config"
	"github.com/gotracker/gotracker/internal/profiling"
)

type profilerCfg struct {
	Profiler            bool   `pflag:"profiler" env:"profiler" usage:"enable profiler (and supporting http server)"`
	ProfilerBindAddress string `pflag:"profiler-bind-addr" env:"profiler_bind_addr" usage:"profiler bind address (if enabled)"`
}

// flags
var profilerConfig = config.NewConfig(profilerCfg{
	Profiler:            false,
	ProfilerBindAddress: "localhost:6060",
})

var rootCmd = &cobra.Command{
	Use:   "gotracker",
	Short: "Gotracker is a tracked music player",
	Long:  `Gotracker is a tracked music player written entirely in Go`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cfg := profilerConfig.Get(); cfg.Profiler {
			profiling.Activate(cfg.ProfilerBindAddress)
		}
		return nil
	},
	Args: cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Usage()
	},
}

func Execute() {
	args := os.Args[1:]
	cmd, _, err := rootCmd.Find(args)
	cmdExec := rootCmd
	if err != nil || (cmd == rootCmd && len(os.Args) > 1) {
		// assume play command if command argument not provided
		cmdExec = playCmd
		os.Args = append([]string{os.Args[0], "play"}, args...)
	}

	if err := cmdExec.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	if profiling.Allowed {
		if err := profilerConfig.Overlay(config.StandardOverlays...).Update(rootCmd); err != nil {
			panic(err)
		}
	}
}
