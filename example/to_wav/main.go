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

	if err := byrd.ConvertMP3FileToWAV(os.Args[1], os.Args[2]); err != nil {
		log.Fatal(err)
	}
}
