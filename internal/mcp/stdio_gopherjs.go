//go:build js

package mcp

import (
	"context"
	"strings"

	"github.com/gopherjs/gopherjs/js"
)

func Run(ctx context.Context, server *Server) error {
	process := js.Global.Get("process")
	stdin := process.Get("stdin")
	stdout := process.Get("stdout")

	done := make(chan struct{})
	lines := make(chan string, 256)
	buffer := ""

	go func() {
		for line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			responses, err := server.Handle(ctx, []byte(line))
			if err != nil {
				stdout.Call("write", err.Error()+"\n")
				continue
			}
			for _, resp := range responses {
				stdout.Call("write", string(resp)+"\n")
			}
		}
		close(done)
	}()

	enqueueLine := func(line string) {
		go func() {
			lines <- line
		}()
	}

	stdin.Call("setEncoding", "utf8")
	stdin.Call("on", "data", func(chunk string) {
		buffer += chunk
		for {
			idx := strings.IndexByte(buffer, '\n')
			if idx < 0 {
				break
			}
			line := buffer[:idx]
			buffer = buffer[idx+1:]
			enqueueLine(line)
		}
	})
	stdin.Call("on", "end", func() {
		if strings.TrimSpace(buffer) != "" {
			enqueueLine(buffer)
		}
		go func() {
			close(lines)
		}()
	})
	stdin.Call("resume")

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}
