package shell

import (
	"github.com/Microsoft/go-winio/pkg/guid"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func Test(t *testing.T) {

	dcli := must(client.NewClientWithOpts(client.FromEnv))

	type requirements struct {
		shell *Shell
	}

	setup := func(t *testing.T) requirements {
		shell := NewShell(dcli, startContainer(t), "")
		return requirements{shell}
	}

	t.Run("read write", func(t *testing.T) {
		fs := setup(t).shell.FS

		fs.Write("/tmp/test.txt", strings.NewReader("lorem"))
		got := fs.Read("/tmp/test.txt").String()

		require.Equal(t, "lorem", got)
	})

	t.Run("exists", func(t *testing.T) {

		tests := []struct {
			name string
			path string
			want bool
		}{
			{"existing file", "/tmp/existing-file", true},
			{"existing dir", "/tmp/existing-dir", true},
			{"non-existing file", "/tmp/non-existing-file", false},
		}

		reqs := setup(t)
		reqs.shell.Cmd(nil, "touch", "/tmp/existing-file").Must()
		reqs.shell.Cmd(nil, "mkdir", "/tmp/existing-dir").Must()

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				fs := reqs.shell.FS
				got := fs.Exists(tt.path)
				require.Exactly(t, tt.want, got)
			})
		}
	})
}

func startContainer(t testing.TB) string {

	var container = "hldocker-exec-test-" + must(guid.NewV4()).String()

	cmd := exec.Command("sh", "-c", `
		container="$1"
		docker run --rm -d --name "$container" alpine sleep infinity
	`, "--", container)
	cmd.Stderr = os.Stderr

	must(0, cmd.Run())

	t.Cleanup(func() {
		_ = exec.Command("docker", "rm", "-fv", container).Run()
	})

	return container
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
