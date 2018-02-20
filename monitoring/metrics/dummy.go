package metrics

// Dummy is a dummy stats recorder.
var Dummy = &dummy{}

type dummy struct{}

func (d *dummy) IncResourceDeleteEventQueued()           {}
func (d *dummy) IncResourceAddEventQueued()              {}
func (d *dummy) IncResourceAddEventProcessedSuccess()    {}
func (d *dummy) IncResourceAddEventProcessedError()      {}
func (d *dummy) IncResourceDeleteEventProcessedSuccess() {}
func (d *dummy) IncResourceDeleteEventProcessedError()   {}
