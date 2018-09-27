package ion

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/richardwilkes/ion/event"
	"github.com/richardwilkes/ion/provisioner"
	"github.com/richardwilkes/toolbox/atexit"
	"github.com/richardwilkes/toolbox/errs"
	"github.com/richardwilkes/toolbox/log/logadapter"
)

// Ion provides communication with Electron.
type Ion struct {
	provisioningPath                   string
	logger                             logadapter.Logger
	additionalElectronArchiveRetriever provisioner.ArchiveRetriever
	dispatcher                         *event.Dispatcher
	tcpListener                        net.Listener
	conn                               net.Conn
	ctx                                context.Context
	cancel                             context.CancelFunc
	shutdownOnce                       sync.Once
}

// New creates a new Ion instance, launching Electron.
func New(options ...Option) (*Ion, error) {
	var err error
	ion := &Ion{}
	for _, option := range options {
		option(ion)
	}
	if ion.provisioningPath == "" {
		if ion.provisioningPath, err = os.Executable(); err != nil {
			return nil, errs.Wrap(err)
		}
		ion.provisioningPath = filepath.Dir(ion.provisioningPath)
	}
	if ion.provisioningPath, err = filepath.Abs(ion.provisioningPath); err != nil {
		return nil, errs.Wrap(err)
	}
	if ion.logger == nil {
		ion.logger = &logadapter.Discarder{}
	}
	if err = provisioner.ProvisionElectron(ion.provisioningPath, ion.additionalElectronArchiveRetriever); err != nil {
		return nil, err
	}
	ion.dispatcher = event.NewDispatcher(ion.logger)
	ion.ctx, ion.cancel = context.WithCancel(context.Background())
	if ion.tcpListener, err = net.Listen("tcp", "127.0.0.1:"); err != nil {
		return nil, errs.Wrap(err)
	}
	accepted := make(chan bool)
	go ion.timeoutWaitingForElectron(accepted)
	go ion.waitForElectron(accepted)
	ion.startElectron()
	atexit.Register(ion.Shutdown)
	return ion, nil
}

func (ion *Ion) timeoutWaitingForElectron(accepted chan bool) {
	select {
	case <-accepted:
	case <-time.After(30 * time.Second):
		ion.logger.Error("Timeout waiting for TCP connection from Electron")
		ion.Shutdown()
	}
}

func (ion *Ion) waitForElectron(accepted chan bool) {
	conn, err := ion.tcpListener.Accept()
	if err != nil {
		ion.logger.Error(errs.Wrap(err))
		ion.Shutdown()
		return
	}
	accepted <- true
	ion.conn = conn
	ion.close(ion.tcpListener)
	ion.tcpListener = nil
	go ion.receiver()
}

func (ion *Ion) startElectron() {
	// RAW: Implement
}

func (ion *Ion) receiver() {
	r := bufio.NewReader(ion.conn)
	for {
		if ion.ctx.Err() != nil {
			return
		}
		buffer, err := r.ReadBytes('\n')
		if err != nil {
			// "wsarecv" is the error sent on Windows when the client closes its connection
			if err == io.EOF || strings.Contains(strings.ToLower(err.Error()), "wsarecv:") {
				ion.Shutdown()
				return
			}
			ion.logger.Error(errs.Wrap(err))
		}
		var e event.Event
		if err = json.Unmarshal(bytes.TrimSpace(buffer), &e); err != nil {
			ion.logger.Error(errs.NewWithCause("Invalid event data", err))
		} else {
			ion.dispatcher.Dispatch(&e)
		}
	}
}

func (ion *Ion) send(data interface{}) error {
	d, err := json.Marshal(data)
	if err != nil {
		return errs.Wrap(err)
	}
	if _, err = ion.conn.Write(append(d, '\n')); err != nil {
		return errs.Wrap(err)
	}
	return nil
}

// Shutdown shuts down Ion. This may be called more than once, but only the
// first call has any effect.
func (ion *Ion) Shutdown() {
	ion.shutdownOnce.Do(ion.shutdown)
}

func (ion *Ion) shutdown() {
	ion.dispatcher.Dispatch(&event.Event{Name: event.AppShutdown})
	ion.dispatcher.Shutdown()
	ion.close(ion.conn)
	ion.conn = nil
}

func (ion *Ion) close(closer io.Closer) {
	if closer != nil {
		if err := closer.Close(); err != nil {
			ion.logger.Error(errs.Wrap(err))
		}
	}
}
