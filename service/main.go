package main

import (
	"github.com/z46-dev/go-logger"
)

var log *logger.Logger

func init() {
	log = logger.NewLogger().SetPrefix("[FREEIPA-DAEMON]", logger.BoldGreen)
}

func main() {
	log.Basic("Starting FreeIPA Daemon...")

	defer func() {
		log.Basic("Shutting down FreeIPA Daemon...")
	}()
}
