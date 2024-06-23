package shell

import "bytes"

type Result struct {
	Code           int
	Stdout, Stderr *bytes.Buffer
}
