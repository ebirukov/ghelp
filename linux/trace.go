package linux

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
)

const (
	tracePipe = "/sys/kernel/debug/tracing/trace_pipe"
)

type TracePipeWatcher struct {
	debugCtx       context.Context
	debugCtxCancel context.CancelFunc
	closed         chan struct{}
	output         io.Writer
}

func NewPipeWatcher(output io.Writer) *TracePipeWatcher {
	return &TracePipeWatcher{
		closed: make(chan struct{}),
		output: output,
	}
}

var StdTracePipeWatcher = NewPipeWatcher(os.Stdout)

func (tp *TracePipeWatcher) Stop() {
	if tp.debugCtx != nil {
		tp.debugCtxCancel()
	}

	tp.WaitStop()
}

func (tp *TracePipeWatcher) WaitStop() {
	if tp.closed != nil {
		<-tp.closed
	}
}

func (tp *TracePipeWatcher) Start() error {
	return tp.StartWithCxt(context.Background())
}

func (tp *TracePipeWatcher) StartWithCxt(ctx context.Context) error {
	if _, err := os.Stat(tracePipe); err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		if mntErr := DebugFS.Mount(); mntErr != nil {
			return fmt.Errorf("failed to mount trace_pipe: %v", mntErr)
		}
	}

	f, err := os.Open(tracePipe)
	if err != nil {
		return fmt.Errorf("error opening trace pipe: %w", err)
	}

	tp.debugCtx, tp.debugCtxCancel = context.WithCancel(ctx)

	go func() {
		defer func() {
			f.Close()
			close(tp.closed)
		}()

		<-tp.debugCtx.Done()

		log.Println("closing trace pipe")
	}()

	go io.Copy(tp.output, bufio.NewReader(f))

	return nil
}
