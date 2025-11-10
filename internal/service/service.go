package service

import (
	"context"
	"encoding/json"
	"fmt"

	"L0/internal/cache"
	"L0/internal/db"
	"L0/internal/dto"
	"L0/internal/model"
)

// OrderService инкапсулирует бизнес-логику сервиса заказов.
type OrderService struct {
	db    *db.DB
	cache *cache.Cache
}

func NewOrderService(database *db.DB, cache *cache.Cache) *OrderService {
	return &OrderService{db: database, cache: cache}
}

// WarmCache загружает все заказы из БД и кэширует их.
func (s *OrderService) WarmCache(ctx context.Context) (int, error) {
	orders, err := s.db.GetAllOrders(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for id, raw := range orders {
		normalized, err := normalize(raw)
		if err != nil {
			continue
		}
		s.cache.Set(id, normalized)
		count++
	}
	return count, nil
}

// ProcessIncoming обрабатывает входящее сообщение из очереди.
func (s *OrderService) ProcessIncoming(ctx context.Context, payload []byte) (string, error) {
	order, normalized, err := decode(payload)
	if err != nil {
		return "", err
	}

	if err := s.db.SaveOrder(ctx, order, payload); err != nil {
		return "", err
	}

	s.cache.Set(order.OrderUID, normalized)
	return order.OrderUID, nil
}

// GetByID возвращает заказ из кэша или БД.
func (s *OrderService) GetByID(ctx context.Context, id string) (json.RawMessage, error) {
	if data, ok := s.cache.Get(id); ok {
		return data, nil
	}

	raw, err := s.db.GetOrder(ctx, id)
	if err != nil || raw == nil {
		return nil, err
	}

	normalized, err := normalize(raw)
	if err != nil {
		return nil, err
	}

	s.cache.Set(id, normalized)
	return normalized, nil
}

func decode(raw []byte) (model.Order, json.RawMessage, error) {
	var order model.Order
	if err := json.Unmarshal(raw, &order); err != nil {
		return model.Order{}, nil, fmt.Errorf("invalid json: %w", err)
	}
	if order.OrderUID == "" {
		return model.Order{}, nil, fmt.Errorf("missing order_uid")
	}

	dtoOrder := dto.FromModel(order)
	normalized, err := json.Marshal(dtoOrder)
	if err != nil {
		return model.Order{}, nil, fmt.Errorf("normalize: %w", err)
	}

	return order, normalized, nil
}

func normalize(raw json.RawMessage) (json.RawMessage, error) {
	var order model.Order
	if err := json.Unmarshal(raw, &order); err != nil {
		return nil, err
	}
	dtoOrder := dto.FromModel(order)
	return json.Marshal(dtoOrder)
}
