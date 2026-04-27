//go:build !js

package mcp

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"
)

func Run(ctx context.Context, server *Server) error {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return err
		}
		lineStr := strings.TrimSpace(string(line))
		if lineStr != "" {
			responses, handleErr := server.Handle(ctx, []byte(lineStr))
			if handleErr != nil {
				return handleErr
			}
			for _, resp := range responses {
				if _, err := writer.Write(resp); err != nil {
					return err
				}
				if err := writer.WriteByte('\n'); err != nil {
					return err
				}
			}
			if err := writer.Flush(); err != nil {
				return err
			}
		}
		if err == io.EOF {
			return nil
		}
	}
}
