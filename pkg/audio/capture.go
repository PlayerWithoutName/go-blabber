package audio

import (
	"fmt"
	"github.com/timshannon/go-openal/openal"
)

const (
	frequency    = 96000
	format       = openal.FormatStereo16
	captureSize  = 256
)

type Source struct {
	SourceChannel chan []byte
	ErrorChannel chan error

	device *openal.CaptureDevice
}

func PrepareDefaultSource() (*Source, error) {
	dev := openal.CaptureOpenDevice("", frequency, format, frequency*2)

	return &Source{
		device:dev,
		SourceChannel:make(chan []byte),
		ErrorChannel:make(chan error, 1),
	}, nil
}

func (source *Source) StartCapture() {
	go func() {

		source.device.CaptureStart()
		err := openal.Err()
		if err != nil {
			source.ErrorChannel <- err
		}


		for{
			samples := make([]byte, captureSize*format.SampleSize())

			if source.device.CapturedSamples() >= captureSize {
				source.device.CaptureTo(samples)
				source.SourceChannel <- samples
			}
			err := openal.Err()
			if err != nil {
				fmt.Println("Capture")
				source.ErrorChannel <- err
			}
		}
	}()
}

func (source *Source) Close() {
	source.device.CloseDevice()
}
