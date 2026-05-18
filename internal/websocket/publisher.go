package websocket

import "log/slog"

// Publish sends an event to the hub channel without blocking the caller.
func Publish(sink chan<- Event, event Event, log *slog.Logger) {
	if sink == nil {
		return
	}

	select {
	case sink <- event:
	default:
		if log != nil {
			log.Warn("realtime event channel full, dropping event",
				slog.String("type", event.Type),
			)
		}
	}
}
