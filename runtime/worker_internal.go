package runtime

import (
	"fmt"
	"sync"
)

type WorkerHandler func(*Worker)

type WorkerMessageHandler func(any)

type Worker struct {
	inbox     chan any
	outbox    chan any
	done      chan struct{}
	doneOnce  sync.Once
	callbacks []WorkerMessageHandler
	mu        sync.Mutex
	err       error
}

type SharedMemory struct {
	mu    sync.RWMutex
	slots []int64
}

func NewWorker(handler WorkerHandler) *Worker {
	worker := &Worker{
		inbox:  make(chan any, 32),
		outbox: make(chan any, 32),
		done:   make(chan struct{}),
	}
	go worker.run(handler)
	return worker
}

func (worker *Worker) PostMessage(message any) error {
	if worker == nil {
		return fmt.Errorf("worker is required")
	}
	if worker.closed() {
		return fmt.Errorf("worker is closed")
	}
	select {
	case <-worker.done:
		return fmt.Errorf("worker is closed")
	case worker.inbox <- message:
		return nil
	}
}

func (worker *Worker) Receive() (any, bool) {
	if worker == nil {
		return nil, false
	}
	select {
	case message := <-worker.inbox:
		return message, true
	case <-worker.done:
		return nil, false
	}
}

func (worker *Worker) PostToParent(message any) error {
	if worker == nil {
		return fmt.Errorf("worker is required")
	}
	if worker.closed() {
		return fmt.Errorf("worker is closed")
	}
	select {
	case <-worker.done:
		return fmt.Errorf("worker is closed")
	case worker.outbox <- message:
		worker.dispatch(message)
		return nil
	}
}

func (worker *Worker) OnMessage(handler WorkerMessageHandler) {
	if worker == nil || handler == nil {
		return
	}
	worker.mu.Lock()
	worker.callbacks = append(worker.callbacks, handler)
	worker.mu.Unlock()
}

func (worker *Worker) NextMessage() (any, bool) {
	if worker == nil {
		return nil, false
	}
	select {
	case message := <-worker.outbox:
		return message, true
	case <-worker.done:
		return nil, false
	}
}

func (worker *Worker) Close() {
	if worker == nil {
		return
	}
	worker.doneOnce.Do(func() {
		close(worker.done)
	})
}

func (worker *Worker) closed() bool {
	select {
	case <-worker.done:
		return true
	default:
		return false
	}
}

func (worker *Worker) LastError() error {
	if worker == nil {
		return nil
	}
	worker.mu.Lock()
	defer worker.mu.Unlock()
	return worker.err
}

func NewSharedMemory(size int) *SharedMemory {
	if size < 0 {
		size = 0
	}
	return &SharedMemory{slots: make([]int64, size)}
}

func (memory *SharedMemory) AtomicLoad(index int) (int64, error) {
	if memory == nil {
		return 0, fmt.Errorf("shared memory is required")
	}
	memory.mu.RLock()
	defer memory.mu.RUnlock()
	if index < 0 || index >= len(memory.slots) {
		return 0, fmt.Errorf("shared memory index out of range")
	}
	return memory.slots[index], nil
}

func (memory *SharedMemory) AtomicStore(index int, value int64) error {
	if memory == nil {
		return fmt.Errorf("shared memory is required")
	}
	memory.mu.Lock()
	defer memory.mu.Unlock()
	if index < 0 || index >= len(memory.slots) {
		return fmt.Errorf("shared memory index out of range")
	}
	memory.slots[index] = value
	return nil
}

func (memory *SharedMemory) Length() int {
	if memory == nil {
		return 0
	}
	memory.mu.RLock()
	defer memory.mu.RUnlock()
	return len(memory.slots)
}

func (worker *Worker) run(handler WorkerHandler) {
	defer worker.Close()
	if handler == nil {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			worker.mu.Lock()
			worker.err = fmt.Errorf("worker panic: %v", recovered)
			worker.mu.Unlock()
		}
	}()
	handler(worker)
}

func (worker *Worker) dispatch(message any) {
	worker.mu.Lock()
	callbacks := append([]WorkerMessageHandler(nil), worker.callbacks...)
	worker.mu.Unlock()
	for _, callback := range callbacks {
		callback(message)
	}
}
