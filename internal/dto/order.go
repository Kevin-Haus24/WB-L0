package dto

import "L0/internal/model"

// Order описывает DTO для выдачи заказа наружу.
type Order struct {
	OrderUID          string   `json:"order_uid"`
	TrackNumber       string   `json:"track_number"`
	Entry             string   `json:"entry"`
	Delivery          Delivery `json:"delivery"`
	Payment           Payment  `json:"payment"`
	Items             []Item   `json:"items"`
	Locale            string   `json:"locale"`
	InternalSignature string   `json:"internal_signature"`
	CustomerID        string   `json:"customer_id"`
	DeliveryService   string   `json:"delivery_service"`
	ShardKey          string   `json:"shardkey"`
	SmID              int      `json:"sm_id"`
	DateCreated       string   `json:"date_created"`
	OofShard          string   `json:"oof_shard"`
}

type Delivery struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Zip     string `json:"zip"`
	City    string `json:"city"`
	Address string `json:"address"`
	Region  string `json:"region"`
	Email   string `json:"email"`
}

type Payment struct {
	Transaction  string `json:"transaction"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency"`
	Provider     string `json:"provider"`
	Amount       int    `json:"amount"`
	PaymentDT    int64  `json:"payment_dt"`
	Bank         string `json:"bank"`
	DeliveryCost int    `json:"delivery_cost"`
	GoodsTotal   int    `json:"goods_total"`
	CustomFee    int    `json:"custom_fee"`
}

type Item struct {
	ChrtID      int    `json:"chrt_id"`
	TrackNumber string `json:"track_number"`
	Price       int    `json:"price"`
	Rid         string `json:"rid"`
	Name        string `json:"name"`
	Sale        int    `json:"sale"`
	Size        string `json:"size"`
	TotalPrice  int    `json:"total_price"`
	NmID        int    `json:"nm_id"`
	Brand       string `json:"brand"`
	Status      int    `json:"status"`
}

// FromModel конвертирует доменную модель в DTO.
func FromModel(m model.Order) Order {
	items := make([]Item, len(m.Items))
	for i, it := range m.Items {
		items[i] = Item{
			ChrtID:      it.ChrtID,
			TrackNumber: it.TrackNumber,
			Price:       it.Price,
			Rid:         it.Rid,
			Name:        it.Name,
			Sale:        it.Sale,
			Size:        it.Size,
			TotalPrice:  it.TotalPrice,
			NmID:        it.NmID,
			Brand:       it.Brand,
			Status:      it.Status,
		}
	}

	return Order{
		OrderUID:    m.OrderUID,
		TrackNumber: m.TrackNumber,
		Entry:       m.Entry,
		Delivery: Delivery{
			Name:    m.Delivery.Name,
			Phone:   m.Delivery.Phone,
			Zip:     m.Delivery.Zip,
			City:    m.Delivery.City,
			Address: m.Delivery.Address,
			Region:  m.Delivery.Region,
			Email:   m.Delivery.Email,
		},
		Payment: Payment{
			Transaction:  m.Payment.Transaction,
			RequestID:    m.Payment.RequestID,
			Currency:     m.Payment.Currency,
			Provider:     m.Payment.Provider,
			Amount:       m.Payment.Amount,
			PaymentDT:    m.Payment.PaymentDT,
			Bank:         m.Payment.Bank,
			DeliveryCost: m.Payment.DeliveryCost,
			GoodsTotal:   m.Payment.GoodsTotal,
			CustomFee:    m.Payment.CustomFee,
		},
		Items:             items,
		Locale:            m.Locale,
		InternalSignature: m.InternalSignature,
		CustomerID:        m.CustomerID,
		DeliveryService:   m.DeliveryService,
		ShardKey:          m.ShardKey,
		SmID:              m.SmID,
		DateCreated:       m.DateCreated,
		OofShard:          m.OofShard,
	}
}
