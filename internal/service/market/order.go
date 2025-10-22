package market

import (
	"errors"
	"time"
)

const (
	OrderStatusNew        OrderStatus = "NEW"
	OrderStatusProcessing OrderStatus = "PROCESSING"
	OrderStatusInvalid    OrderStatus = "INVALID"
	OrderStatusProcessed  OrderStatus = "PROCESSED"
)

type OrderStatus string

type Order struct {
	ID         int64       `json:"-"`
	Number     string      `json:"number"`
	Status     OrderStatus `json:"status"`
	Accrual    float64     `json:"accrual,omitempty"`
	UploadedAt time.Time   `json:"uploaded_at"`
	UserID     *int64      `json:"-"`
}

var (
	ErrOrderNotExists            error = errors.New("order not exists")
	ErrOrderAlreadyExists        error = errors.New("order already exists")
	ErrOrderBelongsToAnotherUser error = errors.New("order belongs to another user")
)
