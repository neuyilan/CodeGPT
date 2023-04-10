package main

import (
	"context"
	"github.com/appleboy/CodeGPT/webhook"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func withContextFunc(ctx context.Context, f func()) context.Context {
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(c)

		select {
		case <-ctx.Done():
		case <-c:
			cancel()
			f()
		}
	}()

	return ctx
}

func main() {
	http.HandleFunc("/webhook", webhook.HandleWebhook)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
