package audio

import (
	"fmt"
	"github.com/timshannon/go-openal/openal"
	"gopkg.in/hraban/opus.v2"
)

type Packet struct {
	Source string
	Data   []byte
}

type Sink struct {
	SinkChannel  chan *Packet
	ErrorChannel chan error

	sources map[string]openal.Source

	device  *openal.Device
	context *openal.Context
	decoder *opus.Decoder
}

func PrepareDefaultSink() (*Sink, error) {
	dev := openal.OpenDevice("")

	context := dev.CreateContext()
	context.Activate()
	err := openal.Err()
	if err != nil {
		return nil, err
	}

	dec, err := opus.NewDecoder(frequency, channels)
	if err != nil {
		return nil, err
	}

	return &Sink{
		SinkChannel:  make(chan *Packet),
		ErrorChannel: make(chan error, 1),
		device:       dev,
		context:      context,
		decoder:      dec,
	}, nil
}

func (sink *Sink) StartPlayback() error {
	/*Source := openal.NewSource()
	err := openal.Err()
	if err != nil {
		return err
	}
	Source.SetLooping(false)*/

	buffers := openal.NewBuffers(40)
	err := openal.Err()
	if err != nil {
		return err
	}

	bufferChannel := make(chan openal.Buffer, 40)

	for _, buffer := range buffers {
		bufferChannel <- buffer
	}

	go func() {
		decodedData := make([]int16, captureSize)
		for packet := range sink.SinkChannel {
			source, ok := sink.sources[packet.Source]
			if !ok {
				source = openal.NewSource()
				err := openal.Err()
				if err != nil {
					sink.ErrorChannel <- err
				}
				source.SetLooping(false)
				sink.sources[packet.Source] = source
			}

			_, err := sink.decoder.Decode(packet.Data, decodedData)
			if err != nil {
				sink.ErrorChannel <- err
			}
			//fmt.Printf("%d in\n", len)

			buffer := <-bufferChannel

			buffer.SetDataInt16(format, decodedData, frequency)
			err = openal.Err()
			if err != nil {
				fmt.Println("SetData")
				sink.ErrorChannel <- err
			}

			source.QueueBuffer(buffer)
			err = openal.Err()
			if err != nil {
				fmt.Println("SetBuffer")
				sink.ErrorChannel <- err
			}

			if source.State() != openal.Playing && source.BuffersQueued() > 2 {
				source.Play()
				err = openal.Err()
				if err != nil {
					fmt.Println("Play")
					sink.ErrorChannel <- err
				}
			}

			processed := source.BuffersProcessed()
			if processed > 0 {
				cleanBuffers := make(openal.Buffers, processed)
				source.UnqueueBuffers(cleanBuffers)
				for _, b := range cleanBuffers {
					bufferChannel <- b
				}
			}
			err = openal.Err()
			if err != nil {
				sink.ErrorChannel <- err
			}
		}
	}()

	return nil
}

func (sink *Sink) Close() {
	sink.context.Destroy()
	sink.device.CloseDevice()
}
