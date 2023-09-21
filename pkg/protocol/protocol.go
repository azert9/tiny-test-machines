package protocol

import "time"

const Port = 2992

type SleepRequest struct {
	Duration time.Duration
}

type SleepResponse struct {
}

type ExecRequest struct {
	Args []string
}

type ExecResponse struct {
	ExitCode int
	Stdout   []byte
}

type UploadRequest struct {
	Path string
	Data []byte
}

type UploadResponse struct {
}
