package command

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/gotracker/gotracker/internal/logging"
	"github.com/gotracker/gotracker/internal/output"
	deviceCommon "github.com/gotracker/gotracker/internal/output/device/common"
	"github.com/gotracker/gotracker/internal/play"
	"github.com/gotracker/gotracker/internal/playlist"
	"github.com/gotracker/playback/player/feature"
)

// persistent flags
var (
	playSettings = play.Settings{
		Output: deviceCommon.Settings{
			Channels:          2,
			SamplesPerSecond:  44100,
			BitsPerSample:     16,
			Filepath:          "output.wav",
			SincFilterEnabled: true,
		},
		NumPremixBuffers:       64,
		PanicOnUnhandledEffect: false,
		ITLongChannelOutput:    false,
		ITEnableNNA:            true,
	}
	loopPlaylist         bool = false
	logger               logging.Squelchable
	disableNativeSamples bool = false
	//disablePreconvertSamples bool = false
)

// flags
var (
	loopSong      bool = false
	startingOrder int  = -1
	startingRow   int  = -1
	randomized    bool = false
	startingBpm   int  = -1
	startingTempo int  = -1
)

func init() {
	output.Setup()

	if persistFlags := playCmd.PersistentFlags(); persistFlags != nil {
		persistFlags.IntVarP(&playSettings.Output.SamplesPerSecond, "sample-rate", "s", playSettings.Output.SamplesPerSecond, "sample rate")
		persistFlags.IntVarP(&playSettings.Output.Channels, "channels", "c", playSettings.Output.Channels, "channels")
		persistFlags.IntVarP(&playSettings.Output.BitsPerSample, "bits-per-sample", "b", playSettings.Output.BitsPerSample, "bits per sample")
		persistFlags.IntVar(&playSettings.NumPremixBuffers, "num-buffers", playSettings.NumPremixBuffers, "number of premixed buffers")
		persistFlags.BoolVarP(&loopPlaylist, "loop-playlist", "L", loopPlaylist, "enable playlist loop (only useful in multi-song mode)")
		persistFlags.BoolVar(&playSettings.ITLongChannelOutput, "it-long", playSettings.ITLongChannelOutput, "enable Impulse Tracker long channel display")
		persistFlags.BoolVar(&playSettings.ITEnableNNA, "it-enable-nna", playSettings.ITEnableNNA, "enable Impulse Tracker New Note Actions")
		persistFlags.BoolVarP(&logger.Squelch, "silent", "q", logger.Squelch, "disable non-error logging")
		persistFlags.StringVarP(&playSettings.Output.Name, "output", "O", output.DefaultOutputDeviceName, "output device")
		persistFlags.StringVarP(&playSettings.Output.Filepath, "output-file", "f", playSettings.Output.Filepath, "output filepath")
		persistFlags.BoolVar(&disableNativeSamples, "disable-native-samples", disableNativeSamples, "disable preconversion of samples to native sampling format")
		//persistFlags.BoolVar(&disablePreconvertSamples, "disable-preconvert-samples", disablePreconvertSamples, "disable preconversion of samples to 32-bit floats")
		persistFlags.BoolVar(&playSettings.Output.SincFilterEnabled, "enable-sinc", playSettings.Output.SincFilterEnabled, "sinc output filter")
	}

	registerPlayFlags(playCmd.Flags())

	rootCmd.AddCommand(playCmd)
}

func registerPlayFlags(flags *pflag.FlagSet) {
	if flags == nil {
		return
	}
	flags.IntVarP(&startingOrder, "starting-order", "o", startingOrder, "starting order (<0 = use song/format default)")
	flags.IntVarP(&startingRow, "starting-row", "r", startingRow, "starting row (<0 = use song/format default)")
	flags.BoolVarP(&loopSong, "loop-song", "l", loopSong, "enable pattern loop (only works in single-song mode)")
	flags.BoolVarP(&randomized, "random", "R", randomized, "randomize the playlist")
	flags.IntVarP(&startingBpm, "bpm", "", startingBpm, "starting BPM (<0 = use song/format default)")
	flags.IntVarP(&startingTempo, "tempo", "", startingTempo, "starting Tempo (ticks per row) (<0 = use song/format default)")
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
	pl := playlist.New()
	for _, fn := range args {
		song := playlist.Song{
			Filepath: fn,
		}
		if startingOrder >= 0 {
			song.Start.Order.Set(startingOrder)
		}
		if startingRow >= 0 {
			song.Start.Row.Set(startingRow)
		}
		if startingBpm >= 0 {
			song.BPM.Set(startingBpm)
		}
		if startingTempo >= 0 {
			song.Tempo.Set(startingTempo)
		}
		if len(args) == 1 {
			if loopSong {
				song.Loop.Count = playlist.NewLoopForever()
			} else {
				song.Loop.Count = playlist.NewLoopCount(0)
			}
		}
		pl.Add(song)
	}

	pl.SetLooping(loopPlaylist)
	pl.SetRandomized(randomized)
	return pl, nil
}

func playSongs(pl *playlist.Playlist) (bool, error) {
	var features []feature.Feature
	features = append(features, feature.UseNativeSampleFormat(!disableNativeSamples))

	return play.Playlist(pl, features, &playSettings, &logger)
}
