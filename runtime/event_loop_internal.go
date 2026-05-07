package runtime

import (
	"sort"
	"time"
)

type RuntimeTask func()

type TimerHandle struct {
	ID       int64
	Repeat   bool
	Delay    time.Duration
	Canceled bool
}

type EventLoop struct {
	now        time.Duration
	nextID     int64
	microtasks []RuntimeTask
	timers     map[int64]*scheduledTimer
}

type scheduledTimer struct {
	handle TimerHandle
	due    time.Duration
	task   RuntimeTask
}

func NewEventLoop() *EventLoop {
	return &EventLoop{timers: map[int64]*scheduledTimer{}}
}

func (loop *EventLoop) QueueMicrotask(task RuntimeTask) {
	if loop == nil || task == nil {
		return
	}
	loop.microtasks = append(loop.microtasks, task)
}

func (loop *EventLoop) SetTimeout(task RuntimeTask, delay time.Duration) TimerHandle {
	return loop.scheduleTimer(task, delay, false)
}

func (loop *EventLoop) SetInterval(task RuntimeTask, delay time.Duration) TimerHandle {
	return loop.scheduleTimer(task, delay, true)
}

func (loop *EventLoop) ClearTimer(handle TimerHandle) bool {
	if loop == nil || handle.ID == 0 {
		return false
	}
	if timer, ok := loop.timers[handle.ID]; ok {
		timer.handle.Canceled = true
		delete(loop.timers, handle.ID)
		return true
	}
	return false
}

func (loop *EventLoop) Advance(delta time.Duration) {
	if loop == nil {
		return
	}
	if delta > 0 {
		loop.now += delta
	}
	loop.RunReady()
}

func (loop *EventLoop) RunReady() {
	if loop == nil {
		return
	}
	loop.RunMicrotasks()
	for {
		ready := loop.readyTimers()
		if len(ready) == 0 {
			return
		}
		for _, timer := range ready {
			if timer.task != nil {
				timer.task()
			}
			loop.RunMicrotasks()
			if timer.handle.Repeat && !timer.handle.Canceled {
				timer.due = loop.now + timer.handle.Delay
				loop.timers[timer.handle.ID] = timer
			}
		}
	}
}

func (loop *EventLoop) RunMicrotasks() {
	if loop == nil {
		return
	}
	for len(loop.microtasks) > 0 {
		task := loop.microtasks[0]
		copy(loop.microtasks, loop.microtasks[1:])
		loop.microtasks = loop.microtasks[:len(loop.microtasks)-1]
		if task != nil {
			task()
		}
	}
}

func (loop *EventLoop) Now() time.Duration {
	if loop == nil {
		return 0
	}
	return loop.now
}

func (loop *EventLoop) PendingTimers() int {
	if loop == nil {
		return 0
	}
	return len(loop.timers)
}

func (loop *EventLoop) PendingMicrotasks() int {
	if loop == nil {
		return 0
	}
	return len(loop.microtasks)
}

func (loop *EventLoop) scheduleTimer(task RuntimeTask, delay time.Duration, repeat bool) TimerHandle {
	if loop == nil || task == nil {
		return TimerHandle{}
	}
	if delay < 0 {
		delay = 0
	}
	loop.nextID++
	handle := TimerHandle{ID: loop.nextID, Repeat: repeat, Delay: delay}
	loop.timers[handle.ID] = &scheduledTimer{handle: handle, due: loop.now + delay, task: task}
	return handle
}

func (loop *EventLoop) readyTimers() []*scheduledTimer {
	ready := make([]*scheduledTimer, 0)
	for id, timer := range loop.timers {
		if timer.due <= loop.now {
			ready = append(ready, timer)
			delete(loop.timers, id)
		}
	}
	sort.SliceStable(ready, func(i, j int) bool {
		if ready[i].due == ready[j].due {
			return ready[i].handle.ID < ready[j].handle.ID
		}
		return ready[i].due < ready[j].due
	})
	return ready
}
