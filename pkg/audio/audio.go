package audio

type Audio struct {
	Sink   *Sink
	Source *Source
}

func SetupAudio() (*Audio, error) {
	sink, err := PrepareDefaultSink()
	if err != nil {
		return nil, err
	}

	source, err := PrepareDefaultSource()
	if err != nil {
		return nil, err
	}

	return &Audio{
		Sink:   sink,
		Source: source,
	}, nil
}

func (audio *Audio) StartLoopback() {
	go func() {
		for sample := range audio.Source.SourceChannel {
			audio.Sink.SinkChannel <- &Packet{
				Data:   sample,
				Source: "loopback",
			}
		}
	}()

	for {
		select {
		case err := <-audio.Source.ErrorChannel:
			panic(err)
		case err := <-audio.Sink.ErrorChannel:
			panic(err)
		}
	}
}

func (audio *Audio) Close() {
	audio.Source.Close()
	audio.Sink.Close()
}
