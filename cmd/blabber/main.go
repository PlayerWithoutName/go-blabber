package main

import (
	"context"
	blb "github.com/PlayerWithoutName/go-blabber/internal/blabber"
)

func main() {
	ctx := context.Background()
	blabber, err := blb.NewBlabber(ctx)
	if err != nil {
		panic(err)
	}
	blabber.SetTopic("blabber-test-topic-voice")
	if err != nil {
		panic(err)
	}
	blabber.StartAudio()
	blabber.Close()
}
