package main

import (
	"context"
	"fmt"
	"github.com/azert9/tiny-test-machines/internal/utils"
	"github.com/azert9/tiny-test-machines/pkg/protocol"
	"github.com/mdlayher/vsock"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
)

var lastCID uint32 = 4

func allocateCID() uint32 {
	// TODO: this is not ideal (cid exhaustion + collisions with other cid sources)
	return atomic.AddUint32(&lastCID, 1)
}

func runVM(ctx context.Context, rpcMode bool) error {

	cmd := exec.CommandContext(ctx, "./run")

	if rpcMode {
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, "TTM_MODE=rpc")
		cmd.Env = append(cmd.Env, fmt.Sprintf("TTM_CID=%d", allocateCID()))
	} else {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	cmd.Cancel = func() error {
		return cmd.Process.Signal(syscall.SIGTERM)
	}

	return cmd.Run()
}

// vsockServer accepts connections on a vsocket and sends their file descriptors through a channel.
func vsockServer(task *utils.Task, connections chan<- net.Conn) error {

	listener, err := vsock.Listen(protocol.Port, &vsock.Config{})
	if err != nil {
		return err
	}
	defer utils.CloseOrLog(listener, "warning: error closing vsock listener")

	var shuttingDown atomic.Bool
	task.StartSubtask(func(task *utils.Task) error {
		// closing the listener when the context is canceled
		_ = <-task.Ctx().Done()
		shuttingDown.Store(true)
		utils.CloseOrLog(listener, "warning: error closing vsock listener")
		return nil
	})

	for {

		conn, err := listener.Accept()
		if err != nil {
			if shuttingDown.Load() {
				return nil
			} else {
				return err
			}
		}

		connections <- conn
	}
}

func handleClientConn(task *utils.Task, conn net.Conn, vsockConnections <-chan net.Conn) error {

	defer utils.CloseOrLog(conn, "warning: failed to close connection")

	// Starting a VM in background

	task.StartSubtask(func(task *utils.Task) error {
		log.Println("starting a new VM")
		err := runVM(task.Ctx(), true)
		if err == nil {
			log.Println("VM exited without error")
		} else {
			log.Printf("VM exited with error: %v", err)
		}
		return err
	})

	// Waiting for a VM to connect back

	var vsockConn net.Conn
	select {
	case <-task.Ctx().Done():
		return task.Ctx().Err()
	case vsockConn = <-vsockConnections:
	}
	defer utils.CloseOrLog(vsockConn, "warning: error closing vsocket connection")

	// Relaying data between client and VM

	task.StartSubtask(func(task *utils.Task) error {
		if _, err := io.Copy(vsockConn, conn); err != nil {
			return fmt.Errorf("copying data from client to VM: %v", err)
		}
		return vsockConn.Close()
	})

	task.StartSubtask(func(task *utils.Task) error {
		if _, err := io.Copy(conn, vsockConn); err != nil {
			return fmt.Errorf("copying data from VM to client: %v", err)
		}
		return nil
	})

	// Waiting for subtasks before running deferred functions

	task.WaitSubtasks()

	return nil
}

func run(ctx context.Context) error {

	socketPath, err := protocol.GetServerSocketPath()
	if err != nil {
		return err
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	var closeListenerOnce sync.Once
	var listenerClosed atomic.Bool
	defer func() {
		closeListenerOnce.Do(func() {
			listenerClosed.Store(true)
			if err := listener.Close(); err != nil {
				log.Printf("warning: failed to close listener socket")
			}
		})
	}()

	return utils.RunTask(ctx, func(task *utils.Task) error {

		task.StartSubtask(func(task *utils.Task) error {
			// closing the listener when the context is canceled
			_ = <-task.Ctx().Done()
			closeListenerOnce.Do(func() {
				listenerClosed.Store(true)
				if err := listener.Close(); err != nil {
					log.Printf("warning: failed to close listener socket")
				}
			})
			return nil
		})

		vsockConnections := make(chan net.Conn)
		task.StartSubtask(func(task *utils.Task) error {
			err := vsockServer(task, vsockConnections)
			if err != nil {
				log.Printf("warning: vsocket server: %v", err)
			}
			return err
		})

		log.Printf("listening on %s", socketPath)

		for {

			conn, err := listener.Accept()
			if err != nil {
				if listenerClosed.Load() {
					return nil
				} else {
					return err
				}
			}

			if task.Ctx().Err() != nil {
				return task.Ctx().Err()
			}

			task.StartSubtask(func(task *utils.Task) error {
				if err := handleClientConn(task, conn, vsockConnections); err != nil {
					log.Printf("warning: error handling client connection: %v", err)
				}
				return nil
			})
		}
	})
}

func main() {

	ctx, cancelCtx := context.WithCancel(context.Background())

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("signal received, shutting down (%v)", sig)
		cancelCtx()
	}()

	err := run(ctx)
	if err != nil {
		log.Printf("fatal error: %v", err)
		os.Exit(1)
	}
}
