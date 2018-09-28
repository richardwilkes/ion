package provisioner

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/richardwilkes/toolbox/errs"
	"github.com/richardwilkes/toolbox/xio"
	"github.com/richardwilkes/toolbox/xio/fs"
)

// FromFileSystem attempts to deploy files from a file system.
func FromFileSystem(version, srcRootPath, dstRootPath string, filesystem http.FileSystem, finalizer DeploymentFinalizer) error {
	var crc uint64
	var err error
	s := loadStatus(dstRootPath)
	if s.Version == version {
		if crc, err = crcTree(dstRootPath); err == nil && crc == s.CRC64 {
			return nil
		}
	}
	if err = os.RemoveAll(dstRootPath); err != nil && !os.IsNotExist(err) {
		return errs.Wrap(err)
	}
	if err = os.MkdirAll(dstRootPath, 0755); err != nil {
		return errs.Wrap(err)
	}
	if err = fs.Walk(filesystem, srcRootPath, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		dst, err2 := filepath.Rel(srcRootPath, path)
		if err2 != nil {
			dst = path
		}
		dst = filepath.Join(dstRootPath, dst)
		if info.IsDir() {
			if err := os.MkdirAll(dst, info.Mode().Perm()|0200); err != nil {
				return errs.Wrap(err)
			}
		} else {
			if err := copyFile(filesystem, path, dst); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return errs.Wrap(err)
	}
	if finalizer != nil {
		if err = finalizer(dstRootPath); err != nil {
			return errs.Wrap(err)
		}
	}
	if crc, err = crcTree(dstRootPath); err != nil {
		return err
	}
	s.Version = version
	s.CRC64 = crc
	return s.save(dstRootPath)
}

func copyFile(filesystem http.FileSystem, src, dst string) (err error) {
	var f http.File
	if f, err = filesystem.Open(src); err != nil {
		return errs.Wrap(err)
	}
	defer xio.CloseIgnoringErrors(f)
	var fi os.FileInfo
	fi, err = f.Stat()
	if err != nil {
		return errs.Wrap(err)
	}
	if err = os.MkdirAll(filepath.Dir(dst), 0775); err != nil {
		return errs.Wrap(err)
	}
	var file *os.File
	if file, err = os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fi.Mode().Perm()|0200); err != nil {
		return errs.Wrap(err)
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = errs.Wrap(cerr)
		}
	}()
	if _, err = io.Copy(file, f); err != nil {
		err = errs.Wrap(err)
	}
	return
}
