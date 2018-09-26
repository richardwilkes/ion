package provisioner

import (
	"hash/crc64"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/richardwilkes/toolbox/errs"
)

func crcTree(basePath string) (uint64, error) {
	var crc uint64
	tab := crc64.MakeTable(crc64.ECMA)
	statusPath := StatusPath(basePath)
	if walkErr := filepath.Walk(basePath, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if path == statusPath {
			return nil
		}
		if info.Name() == ".DS_Store" {
			return nil
		}
		crc = crc64.Update(crc, tab, []byte(path))
		mode := info.Mode()
		if mode&os.ModeSymlink != 0 {
			linkDst, err := os.Readlink(path)
			if err != nil {
				return errs.Wrap(err)
			}
			crc = crc64.Update(crc, tab, []byte(linkDst))
		} else if mode.IsRegular() {
			buffer, err := ioutil.ReadFile(path)
			if err != nil {
				return errs.Wrap(err)
			}
			crc = crc64.Update(crc, tab, buffer)
		}
		return nil
	}); walkErr != nil {
		return 0, errs.Wrap(walkErr)
	}
	return crc, nil
}
