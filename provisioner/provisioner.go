package provisioner

import (
	"archive/zip"
	"bytes"
	"os"

	"github.com/richardwilkes/toolbox/errs"
	xzip "github.com/richardwilkes/toolbox/xio/fs/zip"
)

// DeploymentFinalizer is used to make any final adjustments to files deployed
// from an archive before recording their state.
type DeploymentFinalizer func(dstRootPath string) error

// Provision attempts to deploy an archive of files onto the file system.
// 'finalizer' may be nil.
func Provision(version, dstRootPath string, retriever ArchiveRetriever, finalizer DeploymentFinalizer) error {
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
	var data []byte
	if data, err = retriever(); err != nil {
		return errs.Wrap(err)
	}
	var zr *zip.Reader
	if zr, err = zip.NewReader(bytes.NewReader(data), int64(len(data))); err != nil {
		return errs.Wrap(err)
	}
	if err = xzip.Extract(zr, dstRootPath); err != nil {
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
