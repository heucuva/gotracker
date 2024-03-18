package command

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/gotracker/gotracker/internal/config"
	"github.com/gotracker/gotracker/internal/logging"
	"github.com/gotracker/gotracker/internal/output"
	deviceCommon "github.com/gotracker/gotracker/internal/output/device/common"
	"github.com/gotracker/gotracker/internal/play"
	"github.com/gotracker/gotracker/internal/playlist"
	"github.com/gotracker/playback/player/feature"
)

// persistent flags
var playSettings = config.NewConfig(play.Settings{
	NumPremixBuffers:    64,
	ITLongChannelOutput: false,
	ITEnableNNA:         true,
})

var playOutputSettings = config.NewConfig(deviceCommon.Settings{
	Channels:         2,
	SamplesPerSecond: 44100,
	BitsPerSample:    16,
	StereoSeparation: 50, // 50%
	Filepath:         "output.wav",
})

// flags
type playFlagCfg struct {
	LoopSong             bool `flag:"loop-song" env:"loop_song" f:"l" usage:"enable pattern loop (only works in single-song mode)"`
	StartingOrder        int  `flag:"starting-order" env:"starting_order" f:"o" usage:"starting order (<0 = use song/format default)"`
	StartingRow          int  `flag:"starting-row" env:"starting_row" f:"r" usage:"starting row (<0 = use song/format default)"`
	Randomized           bool `flag:"random" env:"random" f:"R" usage:"randomize the playlist"`
	StartingBPM          int  `flag:"bpm" env:"bpm" usage:"starting BPM (<0 = use song/format default)"`
	StartingTempo        int  `flag:"tempo" env:"tempo" usage:"starting Tempo (ticks per row) (<0 = use song/format default)"`
	LoopPlaylist         bool `pflag:"loop-playlist" env:"loop_playlist" pf:"L" usage:"enable playlist loop (only useful in multi-song mode)"`
	DisableNativeSamples bool `pflag:"disable-native-samples" env:"disable_native_samples" usage:"disable preconversion of samples to native sampling format"`
	//DisablePreconvertSamples bool `pflag:"disable-preconvert-samples" env:"disable_preconvert_samples" usage:"disable preconversion of samples to 32-bit floats"`
}

var playFlags = config.NewConfig(playFlagCfg{
	LoopSong:             false,
	StartingOrder:        -1,
	StartingRow:          -1,
	Randomized:           false,
	StartingBPM:          -1,
	StartingTempo:        -1,
	LoopPlaylist:         false,
	DisableNativeSamples: false,
	//DisablePreconvertSamples: false,
})

var logger = config.NewConfig(logging.Squelchable{
	Squelch: false,
})

func init() {
	output.Setup()

	playOutputSettings.Get().Name = output.DefaultOutputDeviceName

	if err := playSettings.Overlay(config.StandardOverlays...).Update(playCmd); err != nil {
		panic(err)
	}

	if err := playOutputSettings.Overlay(config.StandardOverlays...).Update(playCmd); err != nil {
		panic(err)
	}

	if err := logger.Overlay(config.StandardOverlays...).Update(playCmd); err != nil {
		panic(err)
	}

	registerPlayFlags(playCmd)

	rootCmd.AddCommand(playCmd)
}

func registerPlayFlags(cmd *cobra.Command) {
	if cmd == nil {
		return
	}

	if err := playFlags.Overlay(config.StandardOverlays...).Update(cmd); err != nil {
		panic(err)
	}
}

var playCmd = &cobra.Command{
	Use:   "play [flags] <file(s)>",
	Short: "Play a tracked music file using Gotracker",
	Long:  "Play one or more tracked music file(s) using Gotracker.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && args[0] == "help" {
			return rootCmd.Help()
		}
		pl, err := getPlaylist(args)
		if err != nil {
			return err
		}

		playedAtLeastOne, err := playSongs(pl)
		if err != nil {
			return err
		}

		if !playedAtLeastOne {
			return cmd.Usage()
		}

		return nil
	},
}

func getPlaylist(args []string) (*playlist.Playlist, error) {
	if len(args) == 1 {
		pl, err := getPlaylistFromYaml(args[0])
		if err == nil && pl != nil {
			return pl, nil
		}
	}

	return getPlaylistFromArgList(args)
}

func getPlaylistFromYaml(fn string) (*playlist.Playlist, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	pl, err := playlist.ReadYAML(f, filepath.Dir(fn))
	if err != nil {
		return nil, err
	}

	return pl, nil
}

func getPlaylistFromArgList(args []string) (*playlist.Playlist, error) {
	cfg := playFlags.Get()
	pl := playlist.New()
	for _, fn := range args {
		song := playlist.Song{
			Filepath: fn,
		}
		if cfg.StartingOrder >= 0 {
			song.Start.Order.Set(cfg.StartingOrder)
		}
		if cfg.StartingRow >= 0 {
			song.Start.Row.Set(cfg.StartingRow)
		}
		if cfg.StartingBPM >= 0 {
			song.BPM.Set(cfg.StartingBPM)
		}
		if cfg.StartingTempo >= 0 {
			song.Tempo.Set(cfg.StartingTempo)
		}
		if len(args) == 1 {
			if cfg.LoopSong {
				song.Loop.Count = playlist.NewLoopForever()
			} else {
				song.Loop.Count = playlist.NewLoopCount(0)
			}
		}
		pl.Add(song)
	}

	pl.SetLooping(cfg.LoopPlaylist)
	pl.SetRandomized(cfg.Randomized)
	return pl, nil
}

func playSongs(pl *playlist.Playlist) (bool, error) {
	cfg := playFlags.Get()

	var features []feature.Feature
	features = append(features, feature.UseNativeSampleFormat(!cfg.DisableNativeSamples))

	return play.Playlist(pl, features, playSettings.Get(), playOutputSettings.Get(), playDebugSettings.Get(), logger.Get())
}
