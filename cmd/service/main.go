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
	if data, err := database.GetAllOrders(ctx); err != nil {
		log.Printf("warm cache: %v", err) // Логирует предупреждение о невозможности прогрева.
	} else {
		c.LoadAll(data) // Массово загружает заказы в кэш для ускорения дальнейших запросов.
		log.Printf("cache warmed: %d orders", len(data))
	}

	// Подписка на NATS — настройка обработки входящих сообщений.
	sc, sub, err := nats.Subscribe(natsClusterID, natsClientID, natsChannel, func(msg *stan.Msg) {
		order, normalized, err := decodeOrder(msg.Data) // Пытается декодировать сообщение в структуру заказа и нормализованную JSON.
		if err != nil {
			log.Println("skip message:", err)
			return
		}

		if err := database.SaveOrder(ctx, order, normalized); err != nil {
			log.Println("db save:", err)
			return
		}

		c.Set(order.OrderUID, normalized) // Кладёт нормализованный JSON в кэш по ключу заказа.
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

	mux := http.NewServeMux() // Создаёт мультиплексор HTTP-обработчиков.
	// Обработчик отдаёт заказ из кэша, а при промахе подгружает из БД — описание поведения эндпоинта.
	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) { // Регистрирует обработчик запросов вида /orders/{id}.
		id := r.URL.Path[len("/orders/"):] // Извлекает идентификатор заказа из пути URL.
		if id == "" {
			http.Error(w, "missing order id", http.StatusBadRequest)
			return
		}
		if data, ok := c.Get(id); ok { // Пытается получить заказ из кэша.
			w.Header().Set("Content-Type", "application/json") // Устанавливает тип содержимого ответа.
			w.WriteHeader(http.StatusOK)
			w.Write(data) // Возвращает кэшированный JSON заказ.
			return
		}

		raw, err := database.GetOrder(r.Context(), id) // Загружает заказ из БД при промахе кэша.
		if err != nil {
			log.Printf("get order %s: %v", id, err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if raw == nil { // Проверяет, найден ли заказ.
			http.NotFound(w, r) // Отправляет клиенту статус 404, если заказ отсутствует.
			return              // Завершает обработку запроса.
		}

		c.Set(id, raw)                                     // Кладёт загруженный из БД заказ в кэш для последующих запросов.
		w.Header().Set("Content-Type", "application/json") // Настраивает заголовок ответа.
		w.WriteHeader(http.StatusOK)
		w.Write(raw) // Возвращает клиенту найденный JSON.
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

// decodeOrder проверяет входящий JSON и возвращает нормализованную структуру заказа
func decodeOrder(raw []byte) (model.Order, json.RawMessage, error) {
	var order model.Order
	if err := json.Unmarshal(raw, &order); err != nil { // пытаемся распарсить входящий JSON
		return model.Order{}, nil, fmt.Errorf("invalid json: %w", err)
	}
	if order.OrderUID == "" { // проверяем обязательное поле идентификатора заказа
		return model.Order{}, nil, fmt.Errorf("missing order_uid")
	}

	normalized, err := json.Marshal(order) // сериализуем полученную структуру обратно в JSON, чтобы выровнять формат
	if err != nil {
		return model.Order{}, nil, fmt.Errorf("normalize: %w", err)
	}

	return order, normalized, nil
}
