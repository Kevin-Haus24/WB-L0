package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"L0/internal/cache"
	"L0/internal/db"
	"L0/internal/model"
	"L0/internal/nats"

	stan "github.com/nats-io/stan.go"
)

const (
	dbURL          = "postgres://demo_user:demo_pass@localhost:5432/demo_orders"
	natsClusterID  = "test-cluster"
	natsClientID   = "service-1"
	natsChannel    = "orders"
	httpListenAddr = ":8080"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Подключение к БД
	database, err := db.New(dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer database.Close()

	if err := database.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	// Восстановление кэша из БД
	c := cache.New()
	if data, err := database.GetAllOrders(ctx); err != nil {
		log.Printf("warm cache: %v", err)
	} else {
		c.LoadAll(data)
		log.Printf("cache warmed: %d orders", len(data))
	}

	// Подписка на NATS
	sc, sub, err := nats.Subscribe(natsClusterID, natsClientID, natsChannel, func(msg *stan.Msg) {
		order, normalized, err := decodeOrder(msg.Data)
		if err != nil {
			log.Println("skip message:", err)
			return
		}

		if err := database.SaveOrder(ctx, order, normalized); err != nil {
			log.Println("db save:", err)
			return
		}

		c.Set(order.OrderUID, normalized)
		log.Println("saved order:", order.OrderUID)
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := sub.Close(); err != nil {
			log.Printf("nats close subscription: %v", err)
		}
		sc.Close()
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/orders/"):]
		if id == "" {
			http.Error(w, "missing order id", http.StatusBadRequest)
			return
		}
		if data, ok := c.Get(id); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(data)
			return
		}

		raw, err := database.GetOrder(r.Context(), id)
		if err != nil {
			log.Printf("get order %s: %v", id, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if raw == nil {
			http.NotFound(w, r)
			return
		}

		c.Set(id, raw)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(raw)
	})

	mux.Handle("/", http.FileServer(http.Dir("./web/static")))

	srv := &http.Server{
		Addr:         httpListenAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Println("Server started on", httpListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
}

func decodeOrder(raw []byte) (model.Order, json.RawMessage, error) {
	var order model.Order
	if err := json.Unmarshal(raw, &order); err != nil {
		return model.Order{}, nil, fmt.Errorf("invalid json: %w", err)
	}
	if order.OrderUID == "" {
		return model.Order{}, nil, fmt.Errorf("missing order_uid")
	}

	normalized, err := json.Marshal(order)
	if err != nil {
		return model.Order{}, nil, fmt.Errorf("normalize: %w", err)
	}

	return order, normalized, nil
}
