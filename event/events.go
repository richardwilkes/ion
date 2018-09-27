package event

import "strings"

// Event names.
const (
	// AppReady is sent afer Electron has been launched and is ready.
	AppReady = "app.ready"
	// AppShutdown is send when Electron is shutdown.
	AppShutdown = "app.shutdown"
)

// Event is a union of all event types. All events fill out the Name field.
// Events that use other fields will note their usage in their descriptions.
type Event struct {
	Name string `json:"name"`
}

func (e Event) String() string {
	var buffer strings.Builder
	buffer.WriteString("Event: ")
	buffer.WriteString(e.Name)
	return buffer.String()
}
