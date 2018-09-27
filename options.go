package ion

import (
	"github.com/richardwilkes/ion/provisioner"
	"github.com/richardwilkes/toolbox/log/logadapter"
)

// Option for Ion configuration.
type Option func(*Ion)

// Logger sets a logger to use. Defaults to discarding all logging.
func Logger(logger logadapter.Logger) Option {
	return func(ion *Ion) { ion.logger = logger }
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
