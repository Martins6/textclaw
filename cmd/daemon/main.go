package main

import (
	"fmt"
	"os"

	"github.com/textclaw/textclaw/internal/daemon"
)

func main() {
	if err := daemon.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
