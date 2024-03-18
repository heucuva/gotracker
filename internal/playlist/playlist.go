package playlist

import (
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver"
	"gopkg.in/yaml.v2"

	"github.com/heucuva/optional"
)

type Playlist struct {
	songs             []Song
	currentPlayOrder  []int
	lastPlayed        []int
	lastPlayedMaxSize int
	loop              optional.Value[bool]
	randomized        optional.Value[bool]
}

func New() *Playlist {
	p := Playlist{}
	return &p
}

func (p *Playlist) Reset() {
	*p = *New()
}

type yamlPlaylist struct {
	Version string `yaml:"version,omitempty"`
	Songs   []Song `yaml:"list,omitempty"`
}

const yamlPlaylistCurrentVersion string = "1.0"

func ReadYAML(r io.Reader, basepath string) (*Playlist, error) {
	y := yaml.NewDecoder(r)

	pl := yamlPlaylist{}

	if err := y.Decode(&pl); err != nil {
		return nil, err
	}

	c, _ := semver.NewConstraint("<= " + yamlPlaylistCurrentVersion)
	if ver, err := semver.NewVersion(pl.Version); err == nil {
		valid, msgs := c.Validate(ver)
		if !valid {
			for e := range msgs {
				fmt.Fprintln(os.Stderr, e)
			}
			if len(msgs) > 0 {
				return nil, msgs[0]
			}
		}
	}

	p := New()
	for _, s := range pl.Songs {
		s.Filepath = filepath.Join(basepath, s.Filepath)
		if s.End.Order.IsSet() {
			if !s.End.Row.IsSet() {
				s.End.Row.Set(0) // assume first row of order
			}
		}
		p.Add(s)
	}

	return p, nil
}

func (p *Playlist) WriteYAML(w io.Writer) error {
	y := yaml.NewEncoder(w)
	defer y.Close()

	pl := yamlPlaylist{
		Version: yamlPlaylistCurrentVersion,
		Songs:   p.songs,
	}

	return y.Encode(&pl)
}

func (p *Playlist) Add(s Song) {
	p.songs = append(p.songs, s)
	p.currentPlayOrder = append(p.currentPlayOrder, len(p.songs)-1)
	p.lastPlayedMaxSize = int(math.Floor(float64(len(p.songs)) / (2 * math.Sqrt2)))
}

func (p *Playlist) SetLooping(value bool) {
	p.loop.Set(value)
}

func (p Playlist) IsLooping() bool {
	if v, ok := p.loop.Get(); ok {
		return v
	}
	return false
}

func (p *Playlist) SetRandomized(value bool) {
	p.randomized.Set(value)
}

func (p Playlist) IsRandomized() bool {
	if v, ok := p.randomized.Get(); ok {
		return v
	}
	return false
}

func (p *Playlist) MarkPlayed(s *Song) {
	if !p.IsRandomized() {
		// this is only useful if in randomized mode
		return
	}
	for i := range p.songs {
		if &p.songs[i] == s {
			p.lastPlayed = append(p.lastPlayed, i)
			n := len(p.lastPlayed) - p.lastPlayedMaxSize
			if n > 0 {
				p.lastPlayed = p.lastPlayed[n:]
			}
			return
		}
	}
}

func (p Playlist) GetPlaylist() []int {
	if p.IsRandomized() {
	randomize:
		rand.Shuffle(len(p.currentPlayOrder), func(i, j int) {
			p.currentPlayOrder[j], p.currentPlayOrder[i] = p.currentPlayOrder[i], p.currentPlayOrder[j]
		})
		if len(p.currentPlayOrder) > p.lastPlayedMaxSize && p.lastPlayedMaxSize >= 1 {
			for _, lastIdx := range p.lastPlayed {
				for _, curIdx := range p.currentPlayOrder[:p.lastPlayedMaxSize] {
					if curIdx == lastIdx {
						goto randomize
					}
				}
			}
		}
	}
	return p.currentPlayOrder
}

func (p Playlist) GetSong(idx int) *Song {
	if idx < 0 || idx >= len(p.songs) {
		return nil
	}

	return &p.songs[idx]
}
