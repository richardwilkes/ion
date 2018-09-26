package provisioner

import (
	"path/filepath"

	"github.com/richardwilkes/toolbox/xio/fs"
)

// Status holds information about the current provisioning.
type Status struct {
	Version string
	CRC64   uint64
}

// StatusPath returns the path to the provisioning status file.
func StatusPath(basePath string) string {
	return filepath.Join(basePath, "ion_provisioning.yaml")
}

// LoadStatus loads the provisioning status file.
func LoadStatus(basePath string) *Status {
	var s Status
	if err := fs.LoadYAML(StatusPath(basePath), &s); err != nil {
		return &Status{}
	}
	return &s
}

// Save the provisioning status file.
func (s *Status) Save(basePath string) error {
	return fs.SaveYAML(StatusPath(basePath), s)
}
