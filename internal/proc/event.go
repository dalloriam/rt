package proc

type EventProcessSpawned struct {
	PID uint64 `json:"pid"`
}

func processSpawnedEvent(pid uint64) EventProcessSpawned {
	return EventProcessSpawned{
		PID: pid,
	}
}

type EventProcessExited struct {
	PID   uint64 `json:"pid"`
	Error error  `json:"error"`
}

func processExitedEvent(pid uint64, err error) EventProcessExited {
	return EventProcessExited{
		PID:   pid,
		Error: err,
	}
}
