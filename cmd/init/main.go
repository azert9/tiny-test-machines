package main

import (
	"fmt"
	"github.com/azert9/tiny-test-machines/pkg/protocol"
	"golang.org/x/sys/unix"
	"log"
	"net/rpc"
	"os"
	"os/exec"
	"strings"
	"time"
)

func cmd(args ...string) error {

	cmd := exec.Command(args[0], args[1:]...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func runRPC() error {

	socket, err := unix.Socket(unix.AF_VSOCK, unix.SOCK_STREAM, 0)
	if err != nil {
		return err
	}
	unix.CloseOnExec(socket)

	for {
		addr := unix.SockaddrVM{
			CID:  unix.VMADDR_CID_HOST,
			Port: protocol.Port,
		}
		if err := unix.Connect(socket, &addr); err != nil {
			log.Println(err)
			time.Sleep(time.Second)
			continue
		}
		break
	}

	var vm VM

	srv := rpc.NewServer()

	if err := srv.Register(&vm); err != nil {
		return err
	}

	srv.ServeConn(os.NewFile(uintptr(socket), ""))

	return nil
}

func getKernelArgs() (map[string]string, error) {

	cmdlineBytes, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return nil, err
	}

	cmdline := strings.TrimSpace(string(cmdlineBytes))

	fields := strings.Split(cmdline, " ")

	args := make(map[string]string)
	for _, field := range fields {
		sep := strings.IndexRune(field, '=')
		if sep == -1 {
			args[field] = ""
		} else {
			args[field[:sep]] = field[sep+1:]
		}
	}

	return args, nil
}

func run() error {

	if err := os.Setenv("PATH", "/bin"); err != nil {
		return err
	}

	if err := os.Mkdir("/tmp", 0755); err != nil {
		return err
	}

	if err := os.Mkdir("/sys", 0755); err != nil {
		return err
	}
	if err := cmd("mount", "-t", "sysfs", "sysfs", "/sys"); err != nil {
		return err
	}

	if err := os.Mkdir("/proc", 0755); err != nil {
		return err
	}
	if err := cmd("mount", "-t", "proc", "proc", "/proc"); err != nil {
		return err
	}

	if err := cmd("mknod", "/dev/sda", "b", "8", "0"); err != nil {
		return err
	}

	kernelArgs, err := getKernelArgs()
	if err != nil {
		return err
	}

	vmMode := kernelArgs["ttm-mode"]
	switch vmMode {
	case "":
		fallthrough
	case "shell":
		if err := cmd("sh"); err != nil {
			return err
		}
	case "rpc":
		return runRPC()
	default:
		return fmt.Errorf("invalid mode: %q", vmMode)
	}

	return nil
}

func main() {

	defer cmd("/bin/poweroff", "-f")

	err := run()
	if err != nil {
		log.Printf("fatal error: %v", err)
		os.Exit(1)
	}
}
