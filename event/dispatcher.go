package event

import (
	"sync"

	"github.com/richardwilkes/toolbox/taskqueue"

	"github.com/richardwilkes/toolbox/errs"
	"github.com/richardwilkes/toolbox/log/logadapter"
)

// Listener defines the method necessary to listen for events.
type Listener interface {
	EventFired(event *Event)
}

// Dispatcher provides dispatching of events coming from Electron.
type Dispatcher struct {
	logger    logadapter.ErrorLogger
	queue     *taskqueue.Queue
	lock      sync.RWMutex
	listeners map[string][]Listener
}

// NewDispatcher creates a new dispatcher.
func NewDispatcher(logger logadapter.ErrorLogger) *Dispatcher {
	return &Dispatcher{
		logger: logger,
		queue:  taskqueue.New(taskqueue.Workers(1), taskqueue.Log(logger.Error)),
	}
}

// AddListener adds a listener for the given event names. inFront will cause
// the listener to be added to the front of the list of existing listeners, if
// any.
func (d *Dispatcher) AddListener(listener Listener, inFront bool, eventNames ...string) {
	if listener != nil && len(eventNames) > 0 {
		d.lock.Lock()
		if d.listeners == nil {
			d.listeners = make(map[string][]Listener)
		}
		for _, name := range eventNames {
			list := append(d.listeners[name], listener)
			if inFront {
				copy(list[1:], list[:len(list)-1])
				list[0] = listener
			}
			d.listeners[name] = list
		}
		d.lock.Unlock()
	}
}

// RemoveListener removes a listener for the given event names.
func (d *Dispatcher) RemoveListener(listener Listener, eventNames ...string) {
	if listener != nil && len(eventNames) > 0 {
		d.lock.Lock()
		if d.listeners != nil {
			for _, name := range eventNames {
				list := d.listeners[name]
				switch len(list) {
				case 0:
				case 1:
					if list[0] == listener {
						delete(d.listeners, name)
					}
				default:
					for i, one := range list {
						if one == listener {
							copy(list[i:], list[i+1:])
							list[len(list)-1] = nil
							d.listeners[name] = list[:len(list)-1]
							break
						}
					}
				}
			}
			if len(d.listeners) == 0 {
				d.listeners = nil
			}
		}
		d.lock.Unlock()
	}
}

// Dispatch an event asynchronously. Event delivery is serialized.
func (d *Dispatcher) Dispatch(event *Event) {
	d.queue.Submit(func() {
		for _, listener := range d.listenersForEvent(event) {
			d.dispatchEvent(listener, event)
		}
	})
}

func (d *Dispatcher) listenersForEvent(event *Event) []Listener {
	d.lock.RLock()
	defer d.lock.RUnlock()
	if d.listeners == nil {
		return nil
	}
	listeners := d.listeners[event.Name]
	if len(listeners) == 0 {
		return nil
	}
	list := make([]Listener, len(listeners))
	copy(list, listeners)
	return list
}

func (d *Dispatcher) dispatchEvent(listener Listener, event *Event) {
	defer func() {
		if err := recover(); err != nil {
			d.logger.Error(errs.Newf("recovered from panic in event listener\n%+v", err))
		}
	}()
	listener.EventFired(event)
}

// Shutdown this dispatcher. Does not return until all pending events have
// been dispatched.
func (d *Dispatcher) Shutdown() {
	d.queue.Shutdown()
}
