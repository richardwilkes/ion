package main

import (
	"github.com/richardwilkes/ion"
	"github.com/richardwilkes/ion/event"
	"github.com/richardwilkes/ion/provisioner"
	"github.com/richardwilkes/toolbox/atexit"
	"github.com/richardwilkes/toolbox/log/jot"
	"github.com/richardwilkes/toolbox/xio/fs/embedded"
)

func main() {
	fs := embedded.NewEFS(nil).FileSystem("efs")
	app, err := ion.New(
		ion.Logger(&jot.Logger{}),
		ion.MacOSAppBundleID("com.trollworks.ion_example"),
		ion.ProvisioningPath("support"),
		ion.ElectronArchiveRetriever(provisioner.FileSystemArchiveRetriever(fs, "/"+provisioner.ElectronArchiveName())),
		ion.IconFileSystem(fs),
	)
	jot.FatalIfErr(err)

	app.Dispatcher().AddListener(event.ListenerFunc(func(e *event.Event) { jot.Debug(e) }), false, event.AppShutdown)

	jot.FatalIfErr(app.Start())
	app.Wait()
	atexit.Exit(0)
}
