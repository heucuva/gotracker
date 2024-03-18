package command

import (
	"github.com/gotracker/gotracker/internal/config"
	"github.com/gotracker/gotracker/internal/play"
	"github.com/spf13/cobra"
)

var playDebugSettings = config.NewConfig(play.DebugSettings{
	PanicOnUnhandledEffect: false,
	Tracing:                false,
	TracingFile:            "",
})

func init() {
	if err := playDebugSettings.Overlay(config.StandardOverlays...).Update(playDebugCmd); err != nil {
		panic(err)
	}

	registerPlayFlags(playDebugCmd)

	playCmd.AddCommand(playDebugCmd)
}

var playDebugCmd = &cobra.Command{
	Use:   "debug [flags] <file(s)>",
	Short: "Play a tracked music file using Gotracker (with added debugging)",
	Long:  "Play one or more tracked music file(s) using Gotracker (with added debugging).",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		parent := cmd.Parent()
		return parent.RunE(cmd, args)
	},
}
