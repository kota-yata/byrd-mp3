package main

import (
	"fmt"
	"log"
	"os"

	byrd "github.com/kota-yata/byrd-mp3"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: go run ./example/stream_pcm <input.mp3>")
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	dec, err := byrd.NewDecoder(f)
	if err != nil {
		log.Fatal(err)
	}

	pcm, err := dec.Decode()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("sample_rate=%d\n", pcm.SampleRate)
	fmt.Printf("channels=%d\n", pcm.Channels)
	fmt.Printf("samples=%d\n", len(pcm.Samples))
}
