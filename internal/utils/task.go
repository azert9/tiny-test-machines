package utils

import (
	"context"
	"sync"
)

// Task represents a task running in a goroutine.
type Task struct {
	parent    *Task
	ctx       context.Context
	cancelCtx context.CancelCauseFunc
	wg        sync.WaitGroup
	done      bool
	// err can be read once in the "done" state
	err error
}

func RunTask(ctx context.Context, f func(task *Task) error) error {
	var task Task
	task.ctx, task.cancelCtx = context.WithCancelCause(ctx)
	task.run(f)
	return task.err
}

func (task *Task) Parent() *Task {
	return task.parent
}

func (task *Task) Ctx() context.Context {
	return task.ctx
}

func (task *Task) Cancel(err error) {
	task.cancelCtx(err)
}

// StartSubtask starts a new task that must finish before its parent task.
func (task *Task) StartSubtask(f func(task *Task) error) {

	subtask := Task{
		parent: task,
	}
	subtask.ctx, subtask.cancelCtx = context.WithCancelCause(task.ctx)

	task.wg.Add(1)
	go func() {
		defer task.wg.Done()
		subtask.run(f)
		if subtask.err != nil {
			// TODO
			task.cancelCtx(subtask.err)
		}
	}()
}

func (task *Task) WaitSubtasks() {
	task.wg.Wait()
}

func (task *Task) run(f func(task *Task) error) {

	defer func() {
		if task.err == nil {
			task.wg.Wait()
		}
		task.cancelCtx(task.err)
		task.wg.Wait()
		task.done = true
	}()

	task.err = f(task)
}
