package provisioner

import (
	"path/filepath"

	"github.com/richardwilkes/toolbox/xio/fs"
)

type status struct {
	Version string
	CRC64   uint64
}

func statusPath(rootPath string) string {
	return filepath.Join(rootPath, "ion_provisioning.yaml")
}

func loadStatus(rootPath string) *status {
	var s status
	if err := fs.LoadYAML(statusPath(rootPath), &s); err != nil {
		return &status{}
	}
	return &s
}

func (s *status) save(rootPath string) error {
	return fs.SaveYAML(statusPath(rootPath), s)
}
