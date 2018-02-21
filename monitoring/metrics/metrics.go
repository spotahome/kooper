package metrics

import "time"

// Recorder knows how to record metrics all over the application.
type Recorder interface {
	// IncResourceAddEvent increments in one the metric records of a queued add
	// event in a resource.
	IncResourceAddEventQueued()
	// IncResourceDeleteEvent increments in one the metric records of a queued delete
	// event in a resource.
	IncResourceDeleteEventQueued()
	// IncResourceAddEventProcessedSuccess increments in one the metric records of a
	// processed add event in success.
	IncResourceAddEventProcessedSuccess()
	// IncResourceAddEventProcessedError increments in one the metric records of a
	// processed add event in error.
	IncResourceAddEventProcessedError()
	// IncResourceDeleteEventProcessedSuccess increments in one the metric records of a
	// processed deleteevent in success.
	IncResourceDeleteEventProcessedSuccess()
	// IncResourceDeleteEventProcessedError increments in one the metric records of a
	// processed delete event in error.
	IncResourceDeleteEventProcessedError()
	// ObserveDurationResourceAddEventProcessedSuccess measures the duration it took to process
	// until now a successful processed add event.
	ObserveDurationResourceAddEventProcessedSuccess(start time.Time)
	// ObserveDurationResourceAddEventProcessedError measures the duration it took to process
	// until now a failed processed add event.
	ObserveDurationResourceAddEventProcessedError(start time.Time)
	// ObserveDurationResourceAddEventProcessedSuccess measures the duration it took to process
	// until now a successful processed delete event.
	ObserveDurationResourceDeleteEventProcessedSuccess(start time.Time)
	// ObserveDurationResourceAddEventProcessedError measures the duration it took to process
	// until now a failed processed delete event.
	ObserveDurationResourceDeleteEventProcessedError(start time.Time)
}
