package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"L0/internal/cache"
	"L0/internal/db"
	"L0/internal/nats"
	"L0/internal/service"

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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM) // Создаёт контекст, отменяемый по сигналам SIGINT и SIGTERM.
	defer stop()                                                                           // Обеспечивает отмену уведомлений о сигналах при завершении main.

	// Подключение к БД
	database, err := db.New(dbURL) // Открывает пул соединений к базе
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer database.Close()

	if err := database.EnsureSchema(ctx); err != nil {
		log.Fatalf("ensure schema: %v", err)
	}

	// Восстановление кэша из БД — прогрев оперативного хранилища.
	c := cache.New()
	orders := service.NewOrderService(database, c)
	if warmed, err := orders.WarmCache(ctx); err != nil {
		log.Printf("warm cache: %v", err)
	} else {
		log.Printf("cache warmed: %d orders", warmed)
	}

	// Подписка на NATS — настройка обработки входящих сообщений.
	sc, sub, err := nats.Subscribe(natsClusterID, natsClientID, natsChannel, func(msg *stan.Msg) {
		orderID, err := orders.ProcessIncoming(ctx, msg.Data)
		if err != nil {
			log.Println("skip message:", err)
			return
		}
		log.Println("saved order:", orderID)
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

	mux := http.NewServeMux() // Создаёт мультиплексор HTTP-обработчиков.
	// Обработчик отдаёт заказ из кэша, а при промахе подгружает из БД — описание поведения эндпоинта.
	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) { // Регистрирует обработчик запросов вида /orders/{id}.
		id := r.URL.Path[len("/orders/"):] // Извлекает идентификатор заказа из пути URL.
		if id == "" {
			http.Error(w, "missing order id", http.StatusBadRequest)
			return
		}
		data, err := orders.GetByID(r.Context(), id)
		if err != nil {
			log.Printf("get order %s: %v", id, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if data == nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json") // Настраивает заголовок ответа.
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})

	// Отдаём статический фронт
	mux.Handle("/", http.FileServer(http.Dir("./web/static"))) // Регистрирует файловый сервер для корневого маршрута.

	srv := &http.Server{ // Конструирует HTTP-сервер с заданными параметрами.
		Addr:         httpListenAddr,   // Адрес и порт прослушивания.
		Handler:      mux,              // Мультиплексор, обрабатывающий входящие запросы.
		ReadTimeout:  5 * time.Second,  // Лимит времени чтения запроса.
		WriteTimeout: 10 * time.Second, // Лимит времени отправки ответа.
		IdleTimeout:  60 * time.Second, // Таймаут бездействия для keep-alive соединений.
	}

	// Сервис слушает HTTP в отдельной горутине, чтобы main мог ждать сигнала
	go func() { // Запускает сервер в отдельной горутине.
		log.Println("Server started on", httpListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) { // Запускает слушатель и обрабатывает ошибки, игнорируя штатное закрытие.
			log.Fatalf("http listen: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutdown signal received") // Логирует получение сигнала остановки.

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Создаёт контекст с таймаутом для аккуратного завершения сервера.
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
}
