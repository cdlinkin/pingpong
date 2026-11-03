package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

func Run(
	ctx context.Context,
	server *http.Server,
	shutdownTimeout time.Duration,
) error {
	ch := make(chan error, 1)

	go func() {
		defer close(ch)

		log.Printf("start HTTP server")

		err := server.ListenAndServe()

		if !errors.Is(err, http.ErrServerClosed) {
			ch <- err
		}
	}()

	select {
	case err := <-ch:
		if err != nil {
			return fmt.Errorf(
				"listen and server HTTP: %w",
				err,
			)
		}
	case <-ctx.Done():
		log.Printf("shutdown server...")

		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			shutdownTimeout,
		)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()

			return fmt.Errorf(
				"shutdown server: %w",
				err,
			)
		}

		log.Printf("HTTP server stopped")
	}

	return nil
}
