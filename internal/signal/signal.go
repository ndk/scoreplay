package signal

import (
	"context"
	"errors"
	"os"
	"os/signal"
)

func WaitForSignal(ctx context.Context) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer signal.Stop(c)
	select {
	case sig := <-c:
		return errors.New(sig.String()) //nolint: err113
	case <-ctx.Done():
		return ctx.Err()
	}
}
