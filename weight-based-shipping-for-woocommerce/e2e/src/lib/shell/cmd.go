package shell

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

type Cmd struct {
	driver interface {
		Exec(*Cmd, io.Writer, io.Writer) (int, error)
	}
	ctx   context.Context
	cmd   []string
	Dir   string
	Stdin io.Reader
	User  string
}

func NewCmd(ctx context.Context, name string, args ...string) *Cmd {
	return &Cmd{ctx: ctx, cmd: append([]string{name}, args...)}
}

func (c *Cmd) WithDir(dir string) *Cmd {
	c.Dir = dir
	return c
}

func (c *Cmd) Run() (Result, error) {

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	code, err := c.driver.Exec(c, stdout, stderr)
	if err != nil && code != -1 {
		err = fmt.Errorf("shell command failed; code: %d, stderr: %s", code, stderr.String())
	}

	return Result{code, stdout, stderr}, err
}

func (c *Cmd) Must() Result {
	r, err := c.Run()
	if err != nil {
		panic(err)
	}
	return r
}
