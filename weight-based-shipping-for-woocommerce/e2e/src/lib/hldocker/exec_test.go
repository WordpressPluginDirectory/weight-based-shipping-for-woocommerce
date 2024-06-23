package hldocker

import (
	"context"
	"github.com/Microsoft/go-winio/pkg/guid"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

//goland:noinspection GoUnhandledErrorResult,GoVetLostCancel
func TestExec(t *testing.T) {

	container := startContainer(t)

	type result struct {
		code           int
		stdout, stderr string
	}

	var timer *TTimer
	dcli := createClient()
	exec := func(ctx context.Context, cmd []string, optss ...ExecOpts) (result, error) {
		timer = Timer()
		defer timer.Stop()

		opts := ExecOpts{}
		if len(optss) > 0 {
			opts = optss[0]
		}

		stdout, stderr := strings.Builder{}, strings.Builder{}
		opts.Stdout = &stdout
		opts.Stderr = &stderr

		code, err := Exec(ctx, dcli, container, cmd, opts)

		return result{code, stdout.String(), stderr.String()}, err
	}

	t.Run("example", func(t *testing.T) {
		r, err := exec(bg, []string{"echo", "hello!"})
		require.NoError(t, err)
		require.Exactly(t, result{0, "hello!\n", ""}, r)
	})

	t.Run("non-existing command fails", func(t *testing.T) {
		res, err := exec(bg, []string{"nosuchcommand"})
		_ = res
		require.Error(t, err)
	})

	t.Run("original exit code preserved", func(t *testing.T) {
		res, err := exec(bg, []string{"sh", "-c", "exit 202"})
		require.Error(t, err)
		require.Exactly(t, result{202, "", ""}, res)
	})

	t.Run("handles stdin", func(t *testing.T) {
		res, err := exec(bg, []string{"cat"}, ExecOpts{Stdin: strings.NewReader("lorem")})
		require.NoError(t, err)
		require.Exactly(t, result{0, "lorem", ""}, res)
	})

	t.Run("termination", func(t *testing.T) {

		t.Run("direct binaries without signal handlers terminate immediatelly", func(t *testing.T) {

			ctx, _ := context.WithTimeout(bg, 100*time.Millisecond)
			_, err := exec(ctx, []string{"sleep", "3"})
			require.Error(t, err)
			timer.Require(t, 100, 150)

		})

		t.Run("shell scripts without signal handlers terminate immediatelly", func(t *testing.T) {
			forEachShell(t, func(t *testing.T, shell string) {

				ctx, _ := context.WithTimeout(bg, 100*time.Millisecond)
				_, err := exec(ctx, []string{shell, "-c", "sleep 3"})
				require.Error(t, err)
				timer.Require(t, 100, 150)

			})
		})

		t.Run("shell scripts with signal handlers are allowed to handle it", func(t *testing.T) {
			forEachShell(t, func(t *testing.T, shell string) {

				ctx, _ := context.WithTimeout(bg, 100*time.Millisecond)

				res, err := exec(ctx, []string{shell, "-c", `
					trap 'echo signaled' TERM
					sleep 1
					echo done
				`})

				require.NoError(t, err)
				require.Exactly(t, result{0, "signaled\ndone\n", ""}, res)

				timer.Require(t, 1000, 1100)
			})
		})

		t.Run("all child proccesses are terminated if asked", func(t *testing.T) {
			forEachShell(t, func(t *testing.T, shell string) {

				ctx, _ := context.WithTimeout(bg, 700*time.Millisecond)

				// Note: the following script does not wait for the child processes to finish.
				// Instead, it exits on SIGTERM immediatelly.
				// That is why we have to send the signal to the process group, instead of the main process.

				_, _ = exec(ctx, []string{shell, "-c", `
					echo begin > /tmp/begin
					sh -c 'sleep 0.2; echo first > /tmp/first' &
					sh -c 'sleep 1; echo second > /tmp/second' &
					sh -c 'sleep 1; echo third > /tmp/third' &
					wait
				`}, ExecOpts{Terminate: []Terminate{
					{syscall.SIGTERM, true, 0},
				}})

				time.Sleep(2 * time.Second)

				res, _ := exec(bg, []string{"cat", "/tmp/begin", "/tmp/first", "/tmp/second", "/tmp/third"})
				require.Exactly(t, result{1,
					"begin\nfirst\n",
					"cat: can't open '/tmp/second': No such file or directory\ncat: can't open '/tmp/third': No such file or directory\n",
				}, res)
			})
		})
	})
}

func forEachShell(t *testing.T, f func(t *testing.T, shell string)) {
	for _, shell := range []string{"sh", "bash"} {
		t.Run(shell, func(t *testing.T) {
			f(t, shell)
		})
	}
}

func createClient() *client.Client {
	return must(client.NewClientWithOpts())
}

func startContainer(t testing.TB) string {

	var container = "hldocker-exec-test-" + must(guid.NewV4()).String()

	cmd := exec.Command("sh", "-c", `
		set -e
		dockerfile='
			FROM alpine
			RUN apk add bash
		'
		img="$(echo "$dockerfile" | docker build -q -)"
		container="$1"
		docker run --rm -d --name "$container" "$img" sleep infinity
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

var bg = context.Background()

type TTimer struct {
	started time.Time
	elapsed time.Duration
}

func Timer() *TTimer {
	return &TTimer{started: time.Now()}
}

func (t *TTimer) Stop() *TTimer {
	t.elapsed = time.Since(t.started)
	return t
}

func (t *TTimer) Require(tt *testing.T, min, max int64) {
	tt.Helper()
	if t.elapsed == 0 {
		t.Stop()
	}
	got := t.elapsed.Milliseconds()
	if got < min || got > max {
		require.FailNowf(tt, "invalid duration", "expected duration from %d to %d ms, got %d ms", min, max, got)
	}
}
