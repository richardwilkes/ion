package ion

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/richardwilkes/ion/event"
	"github.com/richardwilkes/ion/provisioner"
	"github.com/richardwilkes/toolbox/atexit"
	"github.com/richardwilkes/toolbox/errs"
	"github.com/richardwilkes/toolbox/log/logadapter"
	"github.com/richardwilkes/toolbox/xio"
)

const ionFSVersion = "1"

//go:generate mkembeddedfs --no-modtime --output ionfs_gen.go --pkg ion --name ionfs --strip ionfs ionfs

// Ion provides communication with Electron.
type Ion struct {
	provisioningPath         string
	macOSAppBundleID         string
	logger                   logadapter.Logger
	electronArchiveRetriever provisioner.ArchiveRetriever
	iconFileSystem           http.FileSystem
	dispatcher               *event.Dispatcher
	tcpListener              net.Listener
	ctx                      context.Context
	cancel                   context.CancelFunc
	shutdownChan             chan bool
	shutdownOnce             sync.Once
	connLock                 sync.RWMutex
	conn                     net.Conn
}

// New creates a new Ion instance, launching Electron.
func New(options ...Option) (*Ion, error) {
	var err error
	ion := &Ion{
		shutdownChan: make(chan bool),
	}
	for _, option := range options {
		option(ion)
	}
	if ion.provisioningPath == "" {
		if ion.provisioningPath, err = os.Executable(); err != nil {
			return nil, errs.Wrap(err)
		}
		ion.provisioningPath = filepath.Join(filepath.Dir(ion.provisioningPath), "support")
	}
	if ion.provisioningPath, err = filepath.Abs(ion.provisioningPath); err != nil {
		return nil, errs.Wrap(err)
	}
	if ion.logger == nil {
		ion.logger = &logadapter.Discarder{}
	}
	if err = provisioner.ProvisionElectron(ion.provisioningPath, ion.macOSAppBundleID, ion.iconFileSystem, ion.electronArchiveRetriever); err != nil {
		return nil, err
	}
	if err = provisioner.FromFileSystem(ionFSVersion, "/", filepath.Join(ion.provisioningPath, "ion"), ionfs.FileSystem("ionfs"), nil); err != nil {
		return nil, err
	}
	ion.dispatcher = event.NewDispatcher(ion.logger)
	atexit.Register(ion.Shutdown)
	return ion, nil
}

// Start Ion.
func (ion *Ion) Start() error {
	ion.ctx, ion.cancel = context.WithCancel(context.Background())
	var err error
	if ion.tcpListener, err = net.Listen("tcp", "127.0.0.1:"); err != nil {
		return errs.Wrap(err)
	}
	accepted := make(chan bool)
	go ion.timeoutWaitingForElectron(accepted)
	go ion.waitForElectron(accepted)
	if err = ion.startElectron(ion.tcpListener.Addr().String()); err != nil {
		ion.cancel()
		return errs.Wrap(err)
	}
	return nil
}

func (ion *Ion) startElectron(addr string) error {
	cmd := exec.CommandContext(ion.ctx, provisioner.ElectronExecutablePath(ion.provisioningPath), filepath.Join(ion.provisioningPath, "ion/ion.js"), addr)
	cmd.Stderr = xio.NewLineWriter(func(data []byte) { ion.logger.Error(provisioner.ElectronName, " stderr: ", string(data)) })
	cmd.Stdout = xio.NewLineWriter(func(data []byte) { ion.logger.Info(provisioner.ElectronName, " stdout: ", string(data)) })
	if err := cmd.Start(); err != nil {
		return errs.Wrap(err)
	}
	go ion.watchElectron(cmd)
	return nil
}

func (ion *Ion) watchElectron(cmd *exec.Cmd) {
	if err := cmd.Wait(); err != nil {
		ion.logger.Errorf("%s: %v", provisioner.ElectronName, err)
	}
	ion.logger.Debug(provisioner.ElectronName + " stopped")
	ion.Shutdown()
}

// Wait until Ion has shutdown. A second call to this will never return.
func (ion *Ion) Wait() {
	<-ion.shutdownChan
}

func (ion *Ion) timeoutWaitingForElectron(accepted chan bool) {
	select {
	case <-accepted:
	case <-time.After(30 * time.Second):
		ion.logger.Error("Timeout waiting for TCP connection from " + provisioner.ElectronName)
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
	ion.connLock.Lock()
	ion.conn = conn
	ion.connLock.Unlock()
	ion.close(ion.tcpListener)
	ion.tcpListener = nil
	go ion.receiver(conn)
}

// Dispatcher returns the dispatcher.
func (ion *Ion) Dispatcher() *event.Dispatcher {
	return ion.dispatcher
}

func (ion *Ion) receiver(r io.Reader) {
	bufferedReader := bufio.NewReader(r)
	for {
		if ion.ctx.Err() != nil {
			return
		}
		buffer, err := bufferedReader.ReadBytes('\n')
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
	d = append(d, '\n')
	ion.connLock.RLock()
	defer ion.connLock.RUnlock()
	if _, err = ion.conn.Write(d); err != nil {
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
	if ion.cancel != nil {
		ion.cancel()
	}
	ion.connLock.Lock()
	defer ion.connLock.Unlock()
	if ion.conn != nil {
		ion.close(ion.conn)
		ion.conn = nil
	}
	close(ion.shutdownChan)
}

func (ion *Ion) close(closer io.Closer) {
	if closer != nil {
		if err := closer.Close(); err != nil {
			ion.logger.Error(errs.Wrap(err))
		}
	}
}
