package provisioner

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/richardwilkes/toolbox/errs"
	"github.com/richardwilkes/toolbox/xio"
	xzip "github.com/richardwilkes/toolbox/xio/fs/zip"
)

// ElectronVersion holds the version of Electron that will be used.
const ElectronVersion = "3.0.0"

func electronOS() string {
	if runtime.GOOS == "windows" {
		return "win32"
	}
	return runtime.GOOS
}

func electronArch() string {
	if runtime.GOARCH == "amd64" {
		return "x64"
	}
	return "ia32"
}

// ElectronDownloadURL returns the URL to use for downloading Electron.
func ElectronDownloadURL() string {
	return fmt.Sprintf("https://github.com/electron/electron/releases/download/v%[1]s/electron-v%[1]s-%[2]s-%[3]s.zip", ElectronVersion, electronOS(), electronArch())
}

// ElectronPath returns the path to the root of the Electron installation.
func ElectronPath(basePath string) string {
	return filepath.Join(basePath, "electron")
}

// Provision attempts to provision Electron.
func Provision(basePath string, client *http.Client) error {
	var crc uint64
	var err error
	electronPath := ElectronPath(basePath)
	status := LoadStatus(electronPath)
	if status.Version == ElectronVersion {
		if crc, err = crcTree(electronPath); err == nil && crc == status.CRC64 {
			return nil
		}
	}
	if err = os.RemoveAll(electronPath); err != nil && !os.IsNotExist(err) {
		return errs.Wrap(err)
	}
	if err = os.MkdirAll(electronPath, 0755); err != nil {
		return errs.Wrap(err)
	}
	var buffer bytes.Buffer
	if err = download(client, ElectronDownloadURL(), &buffer); err != nil {
		return errs.Wrap(err)
	}
	var zr *zip.Reader
	if zr, err = zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len())); err != nil {
		return errs.Wrap(err)
	}
	if err = xzip.Extract(zr, electronPath); err != nil {
		return errs.Wrap(err)
	}
	if crc, err = crcTree(electronPath); err != nil {
		return err
	}
	status.Version = ElectronVersion
	status.CRC64 = crc
	return status.Save(electronPath)
}

func download(client *http.Client, url string, w io.Writer) error {
	resp, err := client.Get(url)
	if err != nil {
		return errs.Wrap(err)
	}
	defer xio.CloseIgnoringErrors(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errs.Newf("Attempted download of %s returned status code %d", url, resp.StatusCode)
	}
	if _, err = io.Copy(w, resp.Body); err != nil {
		return errs.NewfWithCause(err, "Failed to download %s", url)
	}
	return nil
}
