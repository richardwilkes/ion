package event

// Listener defines the method necessary to listen for events.
type Listener interface {
	EventFired(e *Event)
}

// ListenerFunc is an adapter to allow the use of ordinary functions as event
// listeners. If f is a function with the appropriate signature,
// ListenerFunc(f) is a Listener that calls f.
type ListenerFunc func(e *Event)

// EventFired calls f(e).
func (f ListenerFunc) EventFired(e *Event) {
	f(e)
}
