package streamtransport

import (
	"context"
	"io"
	"net"
	"sync"

	"github.com/dogmatiq/dodeca/logging"
	"github.com/jmalloc/harpy"
)

// Serve reads JSON-RPC responses from r and writes their responses to w until
// ctx is canceled or an IO error occurs.
func Serve(
	ctx context.Context,
	x harpy.Exchanger,
	r io.Reader,
	w io.Writer,
	l logging.Logger,
) error {
	el := &harpy.DefaultExchangeLogger{
		Target: l,
	}

	rr := &RequestSetReader{
		Source: r,
	}

	rw := &ResponseWriter{
		Target: w,
	}

	for {
		if err := harpy.Exchange(ctx, x, rr, rw, el); err != nil {
			return err
		}
	}
}

func AcceptAndServe(
	ctx context.Context,
	nl net.Listener,
	x harpy.Exchanger,
	l logging.Logger,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		nl.Close()
	}()

	var g sync.WaitGroup
	defer g.Wait()

	for {
		conn, err := nl.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			return err
		}

		g.Add(1)
		go func() {
			defer g.Done()
			Serve(ctx, x, conn, conn, l)
		}()
	}
}

func ListenAndServe(
	ctx context.Context,
	network, address string,
	x harpy.Exchanger,
	l logging.Logger,
) (err error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return err
	}

	return AcceptAndServe(ctx, listener, x, l)
}
