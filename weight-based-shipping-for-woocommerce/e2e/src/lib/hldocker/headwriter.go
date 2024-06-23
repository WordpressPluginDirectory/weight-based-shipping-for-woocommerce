package hldocker

import (
	"bytes"
	"io"
)

type firstLineWriter struct {
	sink        io.Writer
	onFirstLine func(string)
	firstLine   []byte
}

func grabFirstLine(sink io.Writer, onFirstLine func(string)) *firstLineWriter {
	return &firstLineWriter{
		sink,
		onFirstLine,
		make([]byte, 0, 100),
	}
}

func (hw *firstLineWriter) Write(b []byte) (int, error) {

	var pn int
	if hw.firstLine != nil {

		idx := bytes.IndexByte(b, '\n')
		if idx == -1 {
			hw.firstLine = append(hw.firstLine, b...)
			return len(b), nil
		}

		hw.firstLine = append(hw.firstLine, b[:idx]...)
		hw.onFirstLine(string(hw.firstLine))
		hw.firstLine = nil

		pn = idx + 1
		b = b[idx+1:]
	}

	n, err := hw.sink.Write(b)
	return n + pn, err
}
