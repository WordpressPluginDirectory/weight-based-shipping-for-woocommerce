package hldocker

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"io"
	"log/slog"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

var _ = exec.Cmd{}

type ExecOpts struct {
	User           string
	Stdin          io.Reader
	Stdout, Stderr io.Writer
	Terminate      TermStrategy
}

type TermStrategy = []Terminate

type Terminate struct {
	Signal      syscall.Signal
	SendToGroup bool
	Timeout     time.Duration // zero means wait indefinitely
}

// GentleButInsistent might be useful for programs of unknown nature. It gives the program a chance to terminate
// gracefully, before sending the signal to all child proccesses directly.
var GentleButInsistent = TermStrategy{
	{syscall.SIGTERM, false, 5 * time.Second},
	{syscall.SIGTERM, true, 5 * time.Second},
	{syscall.SIGKILL, true, 0},
}

// ForReliablePrograms is for good programs that shut down gracefully, freeing all their resources and dealing with
// their children and grandchildren.
//
//goland:noinspection GoUnusedGlobalVariable
var ForReliablePrograms = TermStrategy{
	{syscall.SIGTERM, false, 0},
}

// ForShellScripts is for shell one-liners and scripts. You never know if a shell forwards signals to its children and
// grandchildren, and whether it waits for them to stop. This strategy sends the term signal to all child processes.
// It is close to how pressing Ctrl-C in a terminal interrupts all proccesses in the foreground group.
//
//goland:noinspection GoUnusedGlobalVariable
var ForShellScripts = TermStrategy{
	{syscall.SIGTERM, true, 5 * time.Second},
	{syscall.SIGKILL, true, 0},
}

// Killemall is for programs that ignore all signals.
//
//goland:noinspection GoUnusedGlobalVariable
var Killemall = TermStrategy{
	{syscall.SIGKILL, true, 0},
}

func Exec(ctx context.Context, cli *client.Client, container string, cmd []string, opts ExecOpts) (code int, err error) {

	if len(opts.Terminate) == 0 {
		opts.Terminate = GentleButInsistent
	}

	// Note: No need to setsid here since sh creates a new process group that we can signal later with the negative pid.
	// Note: `echo $$` is required to get an internal PID instead of the one returned by the Docker API,
	// which is mostly useless on Mac and Windows, where docker is running in a virtual machine.
	uberCmd := append([]string{
		"sh", "-c",
		`echo $$ && exec "$@"`,
		"--",
	}, cmd...)

	exec, err := cli.ContainerExecCreate(ctx, container, types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true, // attach to both stdout and stderr to wait for the command to finish
		AttachStderr: true,
		Cmd:          uberCmd,
		User:         opts.User,
	})
	if err != nil {
		return -1, fmt.Errorf("exec create: %w", err)
	}

	resp, err := cli.ContainerExecAttach(context.Background(), exec.ID, types.ExecStartCheck{})
	if err != nil {
		return -1, fmt.Errorf("exec attach: %w", err)
	}
	defer resp.Close()

	termCtx, terminate := context.WithCancel(ctx)
	complete := make(chan struct{})
	abort := make(chan struct{})

	opts.Stdout = grabFirstLine(cmp.Or(opts.Stdout, io.Discard), func(firstLine string) {

		pid, err := strconv.Atoi(firstLine)
		if err != nil {
			panic("received non-numeric PID: " + firstLine)
		}

		go termination(termCtx, pid, opts.Terminate, complete, abort, cli, container, slog.With("cmd", cmd))
	})

	inputErrCh := make(chan error, 1)
	go func() {

		var stdinErr error

		if opts.Stdin != nil {
			_, stdinErr = io.Copy(resp.Conn, opts.Stdin)
			if stdinErr != nil {
				terminate()
			}
		}

		err := resp.CloseWrite()
		if err != nil && stdinErr == nil {
			stdinErr = err
		}

		inputErrCh <- err
	}()

	var outputErr error
	go func() {
		outputErr = waitAndReadOutput(resp, opts.Stdout, opts.Stderr)
		close(complete)
	}()

	select {
	case <-abort:
		return -1, fmt.Errorf("aborted")
	case <-complete:
	}

	if outputErr != nil {
		return -1, fmt.Errorf("exec read output: %w", outputErr)
	}

	inputErr := <-inputErrCh
	if inputErr != nil {
		return -1, fmt.Errorf("exec write input: %w", inputErr)
	}

	info, err := cli.ContainerExecInspect(context.Background(), exec.ID)
	if err != nil {
		return 0, fmt.Errorf("exec inspect: %w", err)
	}

	err = nil
	if info.ExitCode != 0 {
		err = fmt.Errorf("%v: non-zero exit code: %d", cmd, info.ExitCode)
	}

	return info.ExitCode, err
}

func termination(
	ctx context.Context,
	pid int, strategy TermStrategy,
	complete <-chan struct{}, abort chan<- struct{},
	cli *client.Client, container string, logger *slog.Logger,
) {

	select {
	case <-complete:
		return
	case <-ctx.Done():
	}

	for _, term := range strategy {

		id := pid
		if term.SendToGroup {
			id = -id
		}

		logger.Debug("sending term signal...")
		if !signal(cli, container, id, term.Signal) {
			logger.Debug("term signal failed")
			return
		}
		logger.Debug("term signal sent")

		var timeout <-chan time.Time
		if term.Timeout > 0 {
			timeout = time.After(term.Timeout)
		}

		select {
		case <-complete:
			logger.Debug("complete")
			return
		case <-timeout:
			logger.Debug("term timeout")
		}
	}

	close(abort)
}

func signal(cli *client.Client, container string, pid int, signal syscall.Signal) bool {

	cmd := []string{"kill", "-" + strconv.Itoa(int(signal)), strconv.Itoa(pid)}

	killOutput := bytes.Buffer{}
	_, err := Exec(context.Background(), cli, container, cmd, ExecOpts{Stdout: &killOutput, Stderr: &killOutput})
	if err != nil {
		slog.Warn("failed to kill exec", "err", err, "output", killOutput.String())
		return false
	}

	return true
}

func waitAndReadOutput(resp types.HijackedResponse, stdout, stderr io.Writer) (err error) {

	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if stdout == io.Discard && stderr == io.Discard {
		_, err = io.Copy(io.Discard, resp.Reader)
		return err
	}

	_, err = stdcopy.StdCopy(stdout, stderr, resp.Reader)
	return err
}
