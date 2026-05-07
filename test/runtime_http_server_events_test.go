package test

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeHTTPServerEventRegistrationAndRequestFlow(t *testing.T) {
	server := jayessruntime.CreateHTTPServer(nil)
	var events []string
	server.On(jayessruntime.HTTPEventListening, func(event jayessruntime.HTTPServerEvent) {
		if event.Address == "" {
			t.Fatalf("listening event should include address")
		}
		events = append(events, event.Name)
	})
	server.On(jayessruntime.HTTPEventConnection, func(event jayessruntime.HTTPServerEvent) {
		events = append(events, event.Name)
	})
	server.On(jayessruntime.HTTPEventRequest, func(event jayessruntime.HTTPServerEvent) {
		events = append(events, event.Name)
		jayessruntime.HTTPStatus(event.Response, http.StatusAccepted)
		jayessruntime.HTTPWriteBody(event.Response, event.Request.Path)
	})
	server.Once(jayessruntime.HTTPEventClose, func(event jayessruntime.HTTPServerEvent) {
		events = append(events, event.Name)
	})

	if err := server.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("Listen returned error: %v", err)
	}
	result, err := jayessruntime.HTTPDoRequest(jayessruntime.HTTPClientRequest{
		URL:     "http://" + server.Addr() + "/node-events",
		Timeout: time.Second,
	})
	if err != nil {
		t.Fatalf("HTTPDoRequest returned error: %v", err)
	}
	if result.StatusCode != http.StatusAccepted || string(result.Body) != "/node-events" {
		t.Fatalf("unexpected HTTP response: %#v body=%q", result, string(result.Body))
	}
	if server.ListenerCount(jayessruntime.HTTPEventRequest) != 1 {
		t.Fatalf("expected one request listener")
	}
	if err := server.Close(nil); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := server.Close(nil); err != nil {
		t.Fatalf("second Close returned error: %v", err)
	}
	requireEventSequence(t, events, []string{
		jayessruntime.HTTPEventListening,
		jayessruntime.HTTPEventConnection,
		jayessruntime.HTTPEventRequest,
		jayessruntime.HTTPEventClose,
	})
}

func TestRuntimeHTTPServerEventEmitterMethods(t *testing.T) {
	server := jayessruntime.CreateHTTPServer(nil)
	var count int
	id := server.On("custom", func(event jayessruntime.HTTPServerEvent) {
		count++
	})
	onceID := server.Once("custom", func(event jayessruntime.HTTPServerEvent) {
		count += 10
	})
	if id == 0 || onceID == 0 {
		t.Fatalf("expected listener ids")
	}
	if got := server.EventNames(); !reflect.DeepEqual(got, []string{"custom"}) {
		t.Fatalf("unexpected event names %v", got)
	}
	server.Emit(jayessruntime.HTTPServerEvent{Name: "custom"})
	server.Emit(jayessruntime.HTTPServerEvent{Name: "custom"})
	if count != 12 {
		t.Fatalf("expected persistent and once listeners to run correctly, got %d", count)
	}
	if !server.Off("custom", id) {
		t.Fatalf("expected listener removal to succeed")
	}
	if server.ListenerCount("custom") != 0 {
		t.Fatalf("expected no custom listeners")
	}
	server.RemoveAllListeners(jayessruntime.HTTPEventRequest)
	if server.ListenerCount(jayessruntime.HTTPEventRequest) != 0 {
		t.Fatalf("expected request listeners to be removed")
	}
}

func TestRuntimeHTTPServerProtocolEvents(t *testing.T) {
	server := jayessruntime.CreateHTTPServer(func(request *jayessruntime.HTTPRequest, response *jayessruntime.HTTPResponse) {
		jayessruntime.HTTPWriteBody(response, "ok")
	})
	var events []string
	for _, name := range []string{
		jayessruntime.HTTPEventCheckContinue,
		jayessruntime.HTTPEventCheckExpectation,
		jayessruntime.HTTPEventConnect,
		jayessruntime.HTTPEventUpgrade,
	} {
		name := name
		server.On(name, func(event jayessruntime.HTTPServerEvent) {
			events = append(events, event.Name)
		})
	}
	server.Emit(jayessruntime.HTTPServerEvent{
		Name:     jayessruntime.HTTPEventCheckContinue,
		Request:  &jayessruntime.HTTPRequest{},
		Response: jayessruntime.NewHTTPResponse(),
	})
	server.Emit(jayessruntime.HTTPServerEvent{Name: jayessruntime.HTTPEventCheckExpectation})
	server.Emit(jayessruntime.HTTPServerEvent{Name: jayessruntime.HTTPEventConnect})
	server.Emit(jayessruntime.HTTPServerEvent{Name: jayessruntime.HTTPEventUpgrade})
	for _, want := range []string{
		jayessruntime.HTTPEventCheckContinue,
		jayessruntime.HTTPEventCheckExpectation,
		jayessruntime.HTTPEventConnect,
		jayessruntime.HTTPEventUpgrade,
	} {
		if !containsEvent(events, want) {
			t.Fatalf("expected protocol event %s in %v", want, events)
		}
	}
}

func requireEventSequence(t *testing.T, events []string, want []string) {
	t.Helper()
	position := 0
	for _, event := range events {
		if position < len(want) && event == want[position] {
			position++
		}
	}
	if position != len(want) {
		t.Fatalf("expected event sequence %v inside %v", want, events)
	}
}

func containsEvent(events []string, want string) bool {
	for _, event := range events {
		if event == want {
			return true
		}
	}
	return false
}
