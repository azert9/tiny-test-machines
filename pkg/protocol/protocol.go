package protocol

import "time"

const Port = 2992

type SleepRequest struct {
	Duration time.Duration
}

type SleepResponse struct {
}
