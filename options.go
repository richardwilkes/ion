package ion

import (
	"net/http"

	"github.com/richardwilkes/ion/provisioner"
	"github.com/richardwilkes/toolbox/log/logadapter"
)

// Option for Ion configuration.
type Option func(*Ion)

// Logger sets a logger to use. Defaults to discarding all logging.
func Logger(logger logadapter.Logger) Option {
	return func(ion *Ion) { ion.logger = logger }
}

// MacOSAppBundleID sets the value to use for the CFBundleIdentifier in plists
// when provisioning.
func MacOSAppBundleID(id string) Option {
	return func(ion *Ion) { ion.macOSAppBundleID = id }
}

// AdditionalElectronArchiveRetriever sets an ArchiveRetriever to use for
// Electron before the default one.
func AdditionalElectronArchiveRetriever(retriever provisioner.ArchiveRetriever) Option {
	return func(ion *Ion) { ion.additionalElectronArchiveRetriever = retriever }
}

// ProvisioningPath sets the provisioning path. The default varies by
// platform.
func ProvisioningPath(path string) Option {
	return func(ion *Ion) { ion.provisioningPath = path }
}

// IconFileSystem sets a file system to use to retrieve icon files for
// provisioning. The files "/app.icns", "/app.png", "/app.ico" may be
// requested, depending upon the platform. It is OK if the file does not
// exist.
func IconFileSystem(fs http.FileSystem) Option {
	return func(ion *Ion) { ion.iconFileSystem = fs }
}
