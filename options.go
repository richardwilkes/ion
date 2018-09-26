package ion

import (
	"github.com/richardwilkes/toolbox/log/logadapter"
)

// Option for Ion configuration.
type Option func(*Ion)

// Logger sets a logger to use. Defaults to discarding all logging.
func Logger(logger logadapter.Logger) Option {
	return func(ion *Ion) { ion.logger = logger }
}
