package utils

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunTask(t *testing.T) {

	funcCalled := false
	err := RunTask(context.Background(), func(task *Task) error {
		funcCalled = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, funcCalled)
}

func TestRunTaskError(t *testing.T) {

	expectedErr := fmt.Errorf("test error")

	err := RunTask(context.Background(), func(task *Task) error {
		return expectedErr
	})

	assert.Equal(t, expectedErr, err)
}

func TestSubtasks(t *testing.T) {

	taskCount := 0

	err := RunTask(context.Background(), func(task *Task) error {

		taskCount++

		task.StartSubtask(func(task *Task) error {
			task.StartSubtask(func(task *Task) error {
				taskCount++
				return nil
			})
			taskCount++
			return nil
		})

		task.StartSubtask(func(task *Task) error {
			taskCount++
			return nil
		})

		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 4, taskCount)
}

func TestTaskErrorCancelsSubtasks(t *testing.T) {

	expectedErr := fmt.Errorf("test error")
	taskCount := 0

	err := RunTask(context.Background(), func(task *Task) error {

		taskCount++

		task.StartSubtask(func(task *Task) error {
			task.StartSubtask(func(task *Task) error {
				_ = <-task.Ctx().Done()
				taskCount++
				return nil
			})
			_ = <-task.Ctx().Done()
			taskCount++
			return nil
		})

		task.StartSubtask(func(task *Task) error {
			_ = <-task.Ctx().Done()
			taskCount++
			return nil
		})

		return expectedErr
	})

	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 4, taskCount)
}

func TestSubtaskErrorCancelsParentTasks(t *testing.T) {

	expectedErr := fmt.Errorf("test error")

	_ = RunTask(context.Background(), func(task *Task) error {

		task.StartSubtask(func(task *Task) error {
			task.StartSubtask(func(task *Task) error {
				return expectedErr
			})
			_ = <-task.Ctx().Done()
			return task.ctx.Err()
		})

		task.StartSubtask(func(task *Task) error {
			_ = <-task.Ctx().Done()
			return nil
		})

		return nil
	})
}
