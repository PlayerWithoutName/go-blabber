package blabber

import (
	"context"
	"fmt"
	au "github.com/PlayerWithoutName/go-blabber/pkg/audio"
	"github.com/PlayerWithoutName/go-blabber/pkg/transport"
	"gx/ipfs/QmcqU6QUDSXprb1518vYDGczrTJTyGwLG9eUa5iNX4xUtS/go-libp2p-peer"
	"time"
)

type Blabber struct {
	transport *transport.Transport
	audio     *au.Audio
	ctx       context.Context

	currentTopic string
}

func NewBlabber(ctx context.Context) (*Blabber, error) {
	tsp, err := transport.Connect(ctx)
	if err != nil {
		return nil, err
	}

	audio, err := au.SetupAudio()
	if err != nil {
		return nil, err
	}

	return &Blabber{
		ctx:       ctx,
		transport: tsp,
		audio:     audio,
	}, nil
}

func (blabber *Blabber) SetTopic(topic string) error {
	blabber.currentTopic = topic
	return blabber.transport.Join(blabber.ctx, topic)
}

func (blabber *Blabber) StartAudio() {
	go func() {
		for buffer := range blabber.audio.Source.SourceChannel{
			blabber.transport.Pubsub.Publish(blabber.currentTopic, buffer)
		}
	}()

	go func() {
		for {
			msg, err := blabber.transport.CurrentTopic.Next(blabber.ctx)
			if err != nil {
				panic(err)
			}

			if peer.ID(msg.From).Pretty() != blabber.transport.Node.ID().Pretty(){
				blabber.audio.Sink.SinkChannel <- msg.Data
			}
		}
	}()

	blabber.audio.Sink.StartPlayback()
	blabber.audio.Source.StartCapture()

	for{
		fmt.Printf("SubPeers: %d, ", len(blabber.transport.Pubsub.ListPeers(blabber.currentTopic)))
		fmt.Printf("Peers: %d\n", len(blabber.transport.Node.Network().Peers()))
		time.Sleep(time.Second)
	}
}

func (blabber *Blabber) Close() {
	blabber.transport.Close()
}
