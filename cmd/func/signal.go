package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// signalContext creates a context that gets cancelled when a SIGINT or SIGTERM
// signal is received.
//
// The context is cancelled on the first received signal. After this, signals
// are not captured and the application terminated immediately on successive
// signals.
func signalContext(ctx context.Context) context.Context {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		s := <-sig

		fmt.Fprintf(os.Stderr, "\nReceived %s signal, cancelling..\n", s)
		cancel()

		fmt.Fprintf(os.Stderr, "Send SIGINT (ctrl-c) to terminate immediately\n")
		signal.Stop(sig)
	}()

	return ctx
}
