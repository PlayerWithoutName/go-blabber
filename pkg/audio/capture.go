package audio

import (
	"fmt"
	"github.com/timshannon/go-openal/openal"
	"gopkg.in/hraban/opus.v2"
)

const (
	frequency   = 48000
	format      = openal.FormatMono16
	captureSize = 480
	bitrate     = 20000
	complexity  = 10
	bandwidth   = opus.Fullband
	channels    = 1
)

type Source struct {
	SourceChannel chan []byte
	ErrorChannel  chan error

	device  *openal.CaptureDevice
	encoder *opus.Encoder
}

func PrepareDefaultSource() (*Source, error) {
	dev := openal.CaptureOpenDevice("", frequency, format, frequency*2)

	enc, err := opus.NewEncoder(frequency, channels, opus.AppVoIP)
	if err != nil {
		return nil, err
	}

	enc.SetBitrate(bitrate)
	enc.SetComplexity(complexity)
	enc.SetMaxBandwidth(bandwidth)
	enc.SetInBandFEC(true)

	return &Source{
		SourceChannel: make(chan []byte),
		ErrorChannel:  make(chan error, 1),
		device:        dev,
		encoder:       enc,
	}, nil
}

func (source *Source) StartCapture() {
	go func() {

		source.device.CaptureStart()
		err := openal.Err()
		if err != nil {
			source.ErrorChannel <- err
		}

		opusBuffer := make([]byte, 100)
		samples := make([]int16, captureSize)
		for {

			if source.device.CapturedSamples() >= captureSize {
				source.device.CaptureToInt16(samples)
				len, err := source.encoder.Encode(samples, opusBuffer)
				if err != nil {
					source.ErrorChannel <- err
				}

				source.SourceChannel <- opusBuffer[:len]
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
