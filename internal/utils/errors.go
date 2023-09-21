package utils

import (
	"io"
	"log"
)

func CloseOrLog(c io.Closer, msg string) {
	err := c.Close()
	if err != nil {
		log.Printf("%s: %v", msg, err)
	}
}
