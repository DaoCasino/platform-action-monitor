package signals

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func CatchStopSignalSimple(ctx context.Context) error {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-signalChannel:
		return fmt.Errorf("received signal: %s", sig)
	case <-ctx.Done():
		return fmt.Errorf("catchStopSignalSimple: context canceled")
	}
}

