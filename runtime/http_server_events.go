package runtime

import (
	"errors"
	"sort"
	"sync"
)

const (
	HTTPEventCheckContinue    = "checkContinue"
	HTTPEventCheckExpectation = "checkExpectation"
	HTTPEventClientError      = "clientError"
	HTTPEventClose            = "close"
	HTTPEventConnect          = "connect"
	HTTPEventConnection       = "connection"
	HTTPEventDropRequest      = "dropRequest"
	HTTPEventError            = "error"
	HTTPEventListening        = "listening"
	HTTPEventRequest          = "request"
	HTTPEventUpgrade          = "upgrade"
)

type HTTPServerEvent struct {
	Name     string
	Server   *HTTPServer
	Request  *HTTPRequest
	Response *HTTPResponse
	Address  string
	Error    error
}

type HTTPServerEventHandler func(HTTPServerEvent)

type httpServerListener struct {
	id      int
	handler HTTPServerEventHandler
	once    bool
}

type httpServerEvents struct {
	mutex     sync.Mutex
	nextID    int
	listeners map[string][]httpServerListener
}

func newHTTPServerEvents() *httpServerEvents {
	return &httpServerEvents{listeners: map[string][]httpServerListener{}}
}

func (events *httpServerEvents) on(name string, handler HTTPServerEventHandler, once bool) int {
	if events == nil || name == "" || handler == nil {
		return 0
	}
	events.mutex.Lock()
	defer events.mutex.Unlock()
	events.nextID++
	listener := httpServerListener{id: events.nextID, handler: handler, once: once}
	events.listeners[name] = append(events.listeners[name], listener)
	return listener.id
}

func (events *httpServerEvents) off(name string, id int) bool {
	if events == nil || name == "" || id == 0 {
		return false
	}
	events.mutex.Lock()
	defer events.mutex.Unlock()
	listeners := events.listeners[name]
	for index, listener := range listeners {
		if listener.id != id {
			continue
		}
		events.listeners[name] = append(listeners[:index], listeners[index+1:]...)
		return true
	}
	return false
}

func (events *httpServerEvents) removeAll(name string) {
	if events == nil {
		return
	}
	events.mutex.Lock()
	defer events.mutex.Unlock()
	if name == "" {
		events.listeners = map[string][]httpServerListener{}
		return
	}
	delete(events.listeners, name)
}

func (events *httpServerEvents) emit(event HTTPServerEvent) bool {
	if events == nil || event.Name == "" {
		return false
	}
	listeners := events.snapshot(event.Name)
	if len(listeners) == 0 {
		return false
	}
	for _, listener := range listeners {
		listener.handler(event)
		if listener.once {
			events.off(event.Name, listener.id)
		}
	}
	return true
}

func (events *httpServerEvents) snapshot(name string) []httpServerListener {
	events.mutex.Lock()
	defer events.mutex.Unlock()
	listeners := events.listeners[name]
	return append([]httpServerListener{}, listeners...)
}

func (events *httpServerEvents) eventNames() []string {
	if events == nil {
		return nil
	}
	events.mutex.Lock()
	defer events.mutex.Unlock()
	names := make([]string, 0, len(events.listeners))
	for name, listeners := range events.listeners {
		if len(listeners) != 0 {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func (events *httpServerEvents) listenerCount(name string) int {
	if events == nil {
		return 0
	}
	events.mutex.Lock()
	defer events.mutex.Unlock()
	return len(events.listeners[name])
}

func (server *HTTPServer) On(name string, handler HTTPServerEventHandler) int {
	return server.events.on(name, handler, false)
}

func (server *HTTPServer) AddListener(name string, handler HTTPServerEventHandler) int {
	return server.On(name, handler)
}

func (server *HTTPServer) Once(name string, handler HTTPServerEventHandler) int {
	return server.events.on(name, handler, true)
}

func (server *HTTPServer) Off(name string, listenerID int) bool {
	return server.events.off(name, listenerID)
}

func (server *HTTPServer) RemoveListener(name string, listenerID int) bool {
	return server.Off(name, listenerID)
}

func (server *HTTPServer) RemoveAllListeners(name string) {
	server.events.removeAll(name)
}

func (server *HTTPServer) Emit(event HTTPServerEvent) bool {
	if event.Server == nil {
		event.Server = server
	}
	return server.events.emit(event)
}

func (server *HTTPServer) EventNames() []string {
	return server.events.eventNames()
}

func (server *HTTPServer) ListenerCount(name string) int {
	return server.events.listenerCount(name)
}

func (server *HTTPServer) emitError(name string, err error) {
	if err == nil {
		return
	}
	server.Emit(HTTPServerEvent{Name: name, Error: err})
	server.Emit(HTTPServerEvent{Name: HTTPEventError, Error: err})
}

func (server *HTTPServer) ensureEvents() error {
	if server == nil {
		return errors.New("http server is nil")
	}
	if server.events == nil {
		server.events = newHTTPServerEvents()
	}
	return nil
}
