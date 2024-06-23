package shell

import (
	"bytes"
	"io"
	"strings"
)

type FS struct {
	shell *Shell
}

func NewShellFS(shell *Shell) *FS {
	return &FS{shell}
}

func (fs *FS) Exists(path string) bool {
	res := fs.shell.ShellCmd(nil, `test -e "$1" && echo "yes" || echo "no"`, path).Must()
	switch strings.TrimSpace(res.Stdout.String()) {
	case "yes":
		return true
	case "no":
		return false
	default:
		panic("file exists check: unexpected output: " + res.Stdout.String())
	}
}

func (fs *FS) Read(path string) *bytes.Buffer {
	return fs.shell.Cmd(nil, "cat", path).Must().Stdout
}

func (fs *FS) Write(path string, contents io.Reader) {
	cmd := fs.shell.Cmd(nil, "sh", "-c", `cat > "$1"`, "--", path)
	cmd.Stdin = contents
	cmd.Must()
}

func (fs *FS) Copy(from, to string) {
	fs.shell.Cmd(nil, "cp", "-r", from, to).Must()
}

func (fs *FS) Remove(path ...string) {
	args := append([]string{"-rf"}, path...)
	fs.shell.Cmd(nil, "rm", args...).Must()
}
