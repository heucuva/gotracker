package format

import (
	"errors"
	"os"

	"gotracker/internal/format/mod"
	"gotracker/internal/format/s3m"
	"gotracker/internal/player/intf"
	"gotracker/internal/player/state"
)

var (
	supportedFormats = make(map[string]intf.Format)
)

// Load loads the a file into a song
func Load(ss *state.Song, filename string) (intf.Format, error) {
	for _, fmt := range supportedFormats {
		if err := fmt.Load(ss, filename); err == nil {
			return fmt, nil
		} else if os.IsNotExist(err) {
			return nil, err
		}
	}
	return nil, errors.New("unsupported format")
}

func init() {
	supportedFormats["s3m"] = s3m.S3M
	supportedFormats["mod"] = mod.MOD
}
