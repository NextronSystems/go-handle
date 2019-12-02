package main

import (
	"log"
	"time"

	"github.com/Codehardt/go-handle"
)

//func init() { handle.DebugWriter(os.Stdout) }

func main() {
	buf := make([]byte, 6000000) // create 6MB buffer
	handles, err := handle.QueryHandles(buf, nil, []handle.HandleType{handle.HandleTypeFile}, time.Second*20)
	if err != nil {
		log.Fatal(err)
	}
	for i, h := range handles {
		if fh, ok := h.(*handle.FileHandle); ok {
			log.Printf("file handle 0x%04X for process %05d with name '%s'", fh.Handle(), fh.Process(), fh.Name())
		} else {
			log.Fatal("no a file handle")
		}
		if i > 50 {
			break
		}
	}
}
