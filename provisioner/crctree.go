package provisioner

import (
	"hash/crc64"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/richardwilkes/toolbox/cmdline"
	"github.com/richardwilkes/toolbox/errs"
)

var crc64Table = crc64.MakeTable(crc64.ECMA)

func crcTree(rootPath string) (uint64, error) {
	crc := crc64.Update(0, crc64Table, []byte(cmdline.AppCmdName))
	statusPath := statusPath(rootPath)
	if walkErr := filepath.Walk(rootPath, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if path == statusPath {
			return nil
		}
		if info.Name() == ".DS_Store" {
			return nil
		}
		crc = crc64.Update(crc, crc64Table, []byte(path))
		mode := info.Mode()
		if mode&os.ModeSymlink != 0 {
			linkDst, err := os.Readlink(path)
			if err != nil {
				return errs.Wrap(err)
			}
			crc = crc64.Update(crc, crc64Table, []byte(linkDst))
		} else if mode.IsRegular() {
			buffer, err := ioutil.ReadFile(path)
			if err != nil {
				return errs.Wrap(err)
			}
			crc = crc64.Update(crc, crc64Table, buffer)
		}
		return nil
	}); walkErr != nil {
		return 0, errs.Wrap(walkErr)
	}
	return crc, nil
}
