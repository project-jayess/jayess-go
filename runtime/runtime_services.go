package runtime

type RuntimeServices struct {
	Loop *EventLoop
}

func NewRuntimeServices() *RuntimeServices {
	return &RuntimeServices{Loop: NewEventLoop()}
}

func (services *RuntimeServices) EventLoop() *EventLoop {
	if services == nil {
		return nil
	}
	if services.Loop == nil {
		services.Loop = NewEventLoop()
	}
	return services.Loop
}
