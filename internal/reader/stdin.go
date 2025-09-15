// Package reader provides input reading functionality for the application.
package reader

import (
	"bufio"
	"context"
	"io"
	"os"
)

// StdinReader reads input from standard input.
type StdinReader struct {
	scanner *bufio.Scanner
}

// NewStdinReader creates a new standard input reader.
func NewStdinReader() *StdinReader {
	return &StdinReader{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// ReadLines reads lines from stdin and sends them to the output channel.
func (r *StdinReader) ReadLines(ctx context.Context, output chan<- string) error {
	defer close(output)
	
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if !r.scanner.Scan() {
				if err := r.scanner.Err(); err != nil && err != io.EOF {
					return err
				}
				return nil
			}
			
			line := r.scanner.Text()
			if line != "" {
				select {
				case output <- line:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
	}
}