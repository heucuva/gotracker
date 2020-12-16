package output

import (
	"gotracker/internal/player/feature"
	"gotracker/internal/player/render"

	"github.com/pkg/errors"
)

var (
	// ErrDeviceNotSupported is returned when the requested device is not supported
	ErrDeviceNotSupported = errors.New("device not supported")
)

// RowOutputFunc defines the callback for when a row is output on the device
type RowOutputFunc func(deviceKind DeviceKind, row render.RowRender)

// Device is an interface to output device operations
type Device interface {
	Play(in <-chan render.RowRender)
	Close()
}

type deviceDetails struct {
	create         createOutputDeviceFunc
	kind           DeviceKind
	priority       devicePriority
	featureDisable []feature.Feature
}

var (
	// DefaultOutputDeviceName is the default device name
	DefaultOutputDeviceName = "none"

	deviceMap = make(map[string]deviceDetails)
)

// CreateOutputDevice creates an output device based on the provided settings
func CreateOutputDevice(settings Settings) (Device, []feature.Feature, error) {
	if details, ok := deviceMap[settings.Name]; ok && details.create != nil {
		dev, err := details.create(settings)
		if err != nil {
			return nil, nil, err
		}
		return dev, details.featureDisable, nil
	}

	return nil, nil, errors.Wrap(ErrDeviceNotSupported, settings.Name)
}

type device struct {
	Device

	onRowOutput RowOutputFunc
}

// Settings is the settings for configuring an output device
type Settings struct {
	Name             string
	Channels         int
	SamplesPerSecond int
	BitsPerSample    int
	Filepath         string
	OnRowOutput      RowOutputFunc
}

func calculateOptimalDefaultOutputDeviceName() string {
	preferredPriority := devicePriority(0)
	preferredName := "none"
	for name, details := range deviceMap {
		if details.priority > preferredPriority {
			preferredName = name
			preferredPriority = details.priority
		}
	}

	return preferredName
}

// Setup finalizes the output device preference system
func Setup() {
	DefaultOutputDeviceName = calculateOptimalDefaultOutputDeviceName()
}
