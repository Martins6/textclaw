package cli

import (
	"os"
	"path/filepath"
)

var (
	homeDir, _     = os.UserHomeDir()
	textclawDir    = filepath.Join(homeDir, ".textclaw")
	textclawSocket = filepath.Join(textclawDir, "textclaw.sock")
)
