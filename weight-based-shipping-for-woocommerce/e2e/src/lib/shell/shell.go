package shell

import (
	"cmp"
	"context"
	"e2e/src/lib/hldocker"
	"fmt"
	"github.com/docker/docker/client"
	"io"
)

type Shell struct {
	dcli      *client.Client
	container string
	user      string
	FS        *FS
}

func NewShell(dockerCli *client.Client, container string, user string) *Shell {

	if container == "" {
		panic("shell: container is not provided")
	}

	shell := &Shell{dockerCli, container, user, nil}

	_, err := shell.Cmd(nil, "true").Run()
	if err != nil {
		panic(fmt.Sprintf("shell: running `true` failed: %s", err))
	}

	shell.FS = NewShellFS(shell)
	return shell
}

func (s *Shell) Cmd(ctx context.Context, name string, args ...string) *Cmd {
	c := NewCmd(ctx, name, args...)
	c.User = s.user
	c.driver = s
	return c
}

func (s *Shell) ShellCmd(ctx context.Context, cmd string, args ...string) *Cmd {
	args = append([]string{"sh", "-c", cmd, "--"}, args...)
	return s.Cmd(ctx, args[0], args[1:]...)
}

func (s *Shell) Exec(cmd *Cmd, stdout, stderr io.Writer) (code int, err error) {

	term := hldocker.ForReliablePrograms
	if cmd.cmd[0] == "sh" || cmd.cmd[0] == "bash" {
		term = hldocker.ForShellScripts
	}

	ctx := cmp.Or(cmd.ctx, context.Background())

	return hldocker.Exec(ctx, s.dcli, s.container, cmd.cmd, hldocker.ExecOpts{
		User:      cmd.User,
		Stdin:     cmd.Stdin,
		Stdout:    stdout,
		Stderr:    stderr,
		Terminate: term,
	})
}
