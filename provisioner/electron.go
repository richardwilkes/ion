package provisioner

import (
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
)

// ElectronVersion holds the version of Electron that will be used.
const ElectronVersion = "3.0.1"

// ElectronPath returns the path to the root of the Electron installation.
func ElectronPath(rootPath string) string {
	return filepath.Join(rootPath, "electron")
}

// ElectronArchiveName returns the name of the Electron archive file.
func ElectronArchiveName() string {
	return fmt.Sprintf("electron-v%s-%s-%s.zip", ElectronVersion, electronOS(), electronArch())
}

// ElectronDownloadURL returns the URL to use for downloading Electron.
func ElectronDownloadURL() string {
	return fmt.Sprintf("https://github.com/electron/electron/releases/download/v%s/%s", ElectronVersion, ElectronArchiveName())
}

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

// ProvisionElectron attempts to provision Electron. If 'archiveRetriever' is
// not nil, it will be tried before the GitHub retriever.
func ProvisionElectron(rootPath string, archiveRetriever ArchiveRetriever) error {
	r := URLArchiveRetriever(&http.Client{}, ElectronDownloadURL())
	if archiveRetriever != nil {
		r = FallbackArchiveRetriever(archiveRetriever, r)
	}
	return Provision(ElectronVersion, ElectronPath(rootPath), r, nil)
}
