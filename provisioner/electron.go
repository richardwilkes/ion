package provisioner

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/richardwilkes/toolbox/cmdline"
	"github.com/richardwilkes/toolbox/errs"
	"github.com/richardwilkes/toolbox/log/jot"
	"github.com/richardwilkes/toolbox/xio"
)

const (
	// ElectronVersion holds the version of Electron that will be used.
	ElectronVersion = "3.0.2"
	// ElectronName holds the name of Electron.
	ElectronName = "Electron"
)

const (
	appSuffix           = ".app"
	stringMarker        = "<string>"
	contentsName        = "Contents"
	frameworksName      = "Frameworks"
	plistName           = "Info.plist"
	macOSName           = "MacOS"
	electronBundleID    = "com.github.electron"
	electronLowerName   = "electron"
	electronApp         = ElectronName + appSuffix
	helper              = " Helper"
	electronHelper      = ElectronName + helper
	electronHelperApp   = electronHelper + appSuffix
	ehSuffix            = " EH"
	electronHelperEH    = electronHelper + ehSuffix
	electronHelperEHApp = electronHelperEH + appSuffix
	npSuffix            = " NP"
	electronHelperNP    = electronHelper + npSuffix
	electronHelperNPApp = electronHelperNP + appSuffix
)

// ElectronPath returns the path to the root of the Electron installation.
func ElectronPath(rootPath string) string {
	return filepath.Join(rootPath, electronLowerName)
}

// ElectronArchiveName returns the name of the Electron archive file.
func ElectronArchiveName() string {
	return fmt.Sprintf("%s-v%s-%s-%s.zip", electronLowerName, ElectronVersion, electronOS(), electronArch())
}

// ElectronDownloadURL returns the URL to use for downloading Electron.
func ElectronDownloadURL() string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/v%s/%s", electronLowerName, electronLowerName, ElectronVersion, ElectronArchiveName())
}

// ElectronExecutablePath returns the path to the Electron executable.
func ElectronExecutablePath(rootPath string) string {
	electronPath := ElectronPath(rootPath)
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(electronPath, cmdline.AppCmdName+appSuffix, contentsName, macOSName, cmdline.AppCmdName)
	case "windows":
		return filepath.Join(electronPath, cmdline.AppCmdName+".exe")
	default:
		return filepath.Join(electronPath, cmdline.AppCmdName)
	}
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

// ProvisionElectron attempts to provision Electron. 'iconFS' may be nil. If
// 'archiveRetriever' is not nil, it will be tried before the GitHub
// retriever.
func ProvisionElectron(rootPath, macOSAppBundleID string, iconFS http.FileSystem, archiveRetriever ArchiveRetriever) error {
	r := URLArchiveRetriever(&http.Client{}, ElectronDownloadURL())
	if archiveRetriever != nil {
		r = FallbackArchiveRetriever(archiveRetriever, r)
	}
	if macOSAppBundleID == "" {
		macOSAppBundleID = electronBundleID // Default back to no change
	}
	return Provision(ElectronVersion, ElectronPath(rootPath), r, func(dstRootPath string) error {
		return electronDeploymentFinalizer(dstRootPath, macOSAppBundleID, iconFS)
	})
}

func electronDeploymentFinalizer(dstRootPath, macOSAppBundleID string, iconFS http.FileSystem) error {
	if err := electronUpdateIcon(dstRootPath, iconFS); err != nil {
		return err
	}
	if err := electronUpdatePLists(dstRootPath, macOSAppBundleID); err != nil {
		return err
	}
	return electronRenameFiles(dstRootPath)
}

func electronUpdateIcon(dstRootPath string, iconFS http.FileSystem) error {
	jot.Debug("electronUpdateIcon: ", dstRootPath)
	if iconFS != nil {
		if runtime.GOOS == "darwin" {
			jot.Debug("looking for /app.icns")
			if f, err := iconFS.Open("/app.icns"); err == nil {
				jot.Debug("found")
				defer xio.CloseIgnoringErrors(f)
				data, err := ioutil.ReadAll(f)
				if err != nil {
					return errs.Wrap(err)
				}
				if err = ioutil.WriteFile(filepath.Join(dstRootPath, electronApp, contentsName, "Resources", electronLowerName+".icns"), data, 0644); err != nil {
					return errs.Wrap(err)
				}
			}
		}
	}
	return nil
}

func electronUpdatePLists(dstRootPath, macOSAppBundleID string) error {
	if runtime.GOOS == "darwin" {
		contentsDir := filepath.Join(dstRootPath, electronApp, contentsName)
		frameworksDir := filepath.Join(contentsDir, frameworksName)
		lookForElectron := []byte(stringMarker + ElectronName)
		replaceWithAppName := []byte(stringMarker + cmdline.AppCmdName)
		lookForBundleID := []byte(stringMarker + electronBundleID)
		replaceWithBundleID := []byte(stringMarker + macOSAppBundleID)
		for _, path := range []string{
			filepath.Join(contentsDir, plistName),
			filepath.Join(frameworksDir, electronHelperEHApp, contentsName, plistName),
			filepath.Join(frameworksDir, electronHelperNPApp, contentsName, plistName),
			filepath.Join(frameworksDir, electronHelperApp, contentsName, plistName),
		} {
			buffer, err := ioutil.ReadFile(path)
			if err != nil {
				return errs.Wrap(err)
			}
			buffer = bytes.Replace(buffer, lookForElectron, replaceWithAppName, -1)
			buffer = bytes.Replace(buffer, lookForBundleID, replaceWithBundleID, -1)
			if err = ioutil.WriteFile(path, buffer, 0644); err != nil {
				return errs.Wrap(err)
			}
		}
	}
	return nil
}

func electronRenameFiles(dstRootPath string) error {
	type rename struct {
		src string
		dst string
	}
	var list []rename
	switch runtime.GOOS {
	case "darwin":
		appDir := filepath.Join(dstRootPath, cmdline.AppCmdName+appSuffix)
		frameworksDir := filepath.Join(appDir, contentsName, frameworksName)
		helperEH := filepath.Join(frameworksDir, cmdline.AppCmdName+helper+ehSuffix+appSuffix)
		helperNP := filepath.Join(frameworksDir, cmdline.AppCmdName+helper+npSuffix+appSuffix)
		helperPath := filepath.Join(frameworksDir, cmdline.AppCmdName+helper+appSuffix)
		list = []rename{
			{
				src: filepath.Join(dstRootPath, electronApp),
				dst: appDir,
			},
			{
				src: filepath.Join(appDir, contentsName, macOSName, ElectronName),
				dst: filepath.Join(appDir, contentsName, macOSName, cmdline.AppCmdName),
			},
			{
				src: filepath.Join(frameworksDir, electronHelperEHApp),
				dst: helperEH,
			},
			{
				src: filepath.Join(helperEH, contentsName, macOSName, electronHelperEH),
				dst: filepath.Join(helperEH, contentsName, macOSName, cmdline.AppCmdName+helper+ehSuffix),
			},
			{
				src: filepath.Join(frameworksDir, electronHelperNPApp),
				dst: helperNP,
			},
			{
				src: filepath.Join(helperNP, contentsName, macOSName, electronHelperNP),
				dst: filepath.Join(helperNP, contentsName, macOSName, cmdline.AppCmdName+helper+npSuffix)},
			{
				src: filepath.Join(frameworksDir, electronHelperApp),
				dst: helperPath,
			},
			{
				src: filepath.Join(helperPath, contentsName, macOSName, electronHelper),
				dst: filepath.Join(helperPath, contentsName, macOSName, cmdline.AppCmdName+helper),
			},
		}
	case "linux":
		list = []rename{
			{
				src: filepath.Join(dstRootPath, electronLowerName),
				dst: filepath.Join(dstRootPath, cmdline.AppCmdName),
			},
		}
	case "windows":
		list = []rename{
			{
				src: filepath.Join(dstRootPath, electronLowerName+".exe"),
				dst: filepath.Join(dstRootPath, cmdline.AppCmdName+".exe"),
			},
		}
	}
	for _, r := range list {
		if r.src != r.dst {
			if err := os.Rename(r.src, r.dst); err != nil {
				return errs.Wrap(err)
			}
		}
	}
	return nil
}
