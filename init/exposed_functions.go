package main

import (
	"github.com/azert9/tiny-test-machines/pkg/protocol"
	"time"
)

type VM struct {
}

func (v *VM) Sleep(req *protocol.SleepRequest, resp *protocol.SleepResponse) error {
	time.Sleep(req.Duration)
	return nil
}
