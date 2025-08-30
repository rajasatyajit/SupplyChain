package integration

import (
	"os"
	"path/filepath"
	"strconv"
)

// containersAvailable returns true if a Docker or Podman socket is present
func containersAvailable() bool {
	// Docker socket
	if _, err := os.Stat("/var/run/docker.sock"); err == nil {
		return true
	}
	// Podman socket per-user
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		if uid := os.Getuid(); uid > 0 {
			candidate := "/run/user/" + strconv.Itoa(uid) + "/podman/podman.sock"
			if _, err := os.Stat(candidate); err == nil {
				return true
			}
		}
	} else {
		candidate := filepath.Join(runtimeDir, "podman", "podman.sock")
		if _, err := os.Stat(candidate); err == nil {
			return true
		}
	}
	return false
}
