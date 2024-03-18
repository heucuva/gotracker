package common

// Settings is the settings for configuring an output device
type Settings struct {
	Name             string `pflag:"output" env:"output" pf:"O" usage:"output device"`
	Channels         int    `pflag:"channels" env:"channels" pf:"c" usage:"channels"`
	SamplesPerSecond int    `pflag:"sample-rate" env:"sample_rate" pf:"s" usage:"sample rate"`
	BitsPerSample    int    `pflag:"bits-per-sample" env:"bits_per_sample" pf:"b" usage:"bits per sample"`
	StereoSeparation int    `pflag:"stereo-separation" env:"stereo_separation" pf:"S" usage:"stereo separation (0-100)"`
	Filepath         string `pflag:"output-file" env:"-" pf:"f" usage:"output filepath"`
	OnRowOutput      WrittenCallback
}
