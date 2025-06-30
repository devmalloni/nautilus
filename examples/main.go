package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/devmalloni/nautilus"
)

func main() {
	errCh := make(chan error, 100)
	defer close(errCh)
	go func() {
		for err := range errCh {
			log.Printf("Error: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	n := nautilus.New(
		nautilus.WithPersister(nautilus.NewInMemoryPersister()),
		nautilus.WithJsonSchemaValidator(nautilus.NewStandardJsonSchemaValidator()),
		nautilus.WithHttpClient(http.DefaultClient),
		nautilus.WithWorkersCount(5),
		nautilus.WithScheduleBufferSize(100),
		nautilus.WithErrCh(errCh))

	err := n.LoadFromYamlFile(context.Background(), "config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	listenHttp(ctx)

	go n.Run(ctx)

	n.MustSchedule(ctx, nautilus.ID("single_id"), "on_created", nautilus.Global, json.RawMessage(`{"entity_id": "example"}`))

	<-time.After(40 * time.Second)
}

func listenHttp(ctx context.Context) {
	server := &http.Server{
		Addr: ":3333",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var response struct {
				EntityID string `json:"entity_id"`
			}

			json.NewDecoder(r.Body).Decode(&response)

			log.Printf("webhook received with id %s", response.EntityID)
		}),
	}

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}

}
