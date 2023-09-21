package protocol

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetServerSocketPath() (string, error) {

	xdgRuntimeDir, xdgRuntimeDirIsSet := os.LookupEnv("XDG_RUNTIME_DIR")
	if !xdgRuntimeDirIsSet {
		return "", fmt.Errorf("XDG_RUNTIME_DIR is not set")
	}

	return filepath.Join(xdgRuntimeDir, "tiny-test-machines.sock"), nil
}
