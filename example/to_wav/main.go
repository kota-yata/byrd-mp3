package main

import (
	"log"
	"os"

	byrd "github.com/kota-yata/byrd-mp3"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("usage: go run ./example/to_wav <input.mp3> <output.wav>")
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

	if err := pcm.WriteWAVFile(os.Args[2]); err != nil {
		log.Fatal(err)
	}
}
