package test

import (
	"reflect"
	"testing"
	"time"

	jayessruntime "jayess-go/runtime"
)

func TestRuntimeEventLoopRunsMicrotasksBeforeTimers(t *testing.T) {
	loop := jayessruntime.NewEventLoop()
	order := []string{}

	loop.SetTimeout(func() {
		order = append(order, "timer")
		loop.QueueMicrotask(func() {
			order = append(order, "timer-microtask")
		})
	}, time.Millisecond)
	loop.QueueMicrotask(func() {
		order = append(order, "microtask")
	})

	loop.Advance(time.Millisecond)
	want := []string{"microtask", "timer", "timer-microtask"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("unexpected event loop order: got %#v want %#v", order, want)
	}
}

func TestRuntimeEventLoopIntervalAndCancellation(t *testing.T) {
	loop := jayessruntime.NewEventLoop()
	count := 0
	handle := loop.SetInterval(func() {
		count++
	}, 10*time.Millisecond)

	loop.Advance(10 * time.Millisecond)
	loop.Advance(10 * time.Millisecond)
	if count != 2 {
		t.Fatalf("expected interval twice, got %d", count)
	}
	if !loop.ClearTimer(handle) {
		t.Fatal("expected interval cancellation")
	}
	loop.Advance(10 * time.Millisecond)
	if count != 2 {
		t.Fatalf("expected canceled interval to stop at 2, got %d", count)
	}
}

func TestRuntimeServicesOwnEventLoop(t *testing.T) {
	services := jayessruntime.NewRuntimeServices()
	if services.EventLoop() == nil {
		t.Fatal("expected runtime services event loop")
	}
	if services.EventLoop() != services.EventLoop() {
		t.Fatal("expected stable runtime services event loop")
	}
}
