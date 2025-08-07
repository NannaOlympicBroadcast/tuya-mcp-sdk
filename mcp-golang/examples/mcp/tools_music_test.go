package mcp

import (
	"log"
	"testing"
	"time"
)

func TestMusic(t *testing.T) {
	music := GetMusic()

	go func() {
		err := music.Play("classic")
		if err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(10 * time.Second)
	music.Stop()

	time.Sleep(10 * time.Second)
	music.Play("classic")

	time.Sleep(10 * time.Second)
	music.Stop()
}
