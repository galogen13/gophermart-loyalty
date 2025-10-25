package market

import (
	"errors"
	"strconv"
	"strings"
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

func OrderNumberLuhnCheck(number string) error {

	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, number)

	if len(cleaned) < 2 {
		return ErrOrderIncorrectNumber
	}

	sum := 0
	isSecond := false

	for i := len(cleaned) - 1; i >= 0; i-- {
		digit, err := strconv.Atoi(string(cleaned[i]))
		if err != nil {
			return ErrOrderIncorrectNumber
		}

		if isSecond {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		isSecond = !isSecond
	}

	if sum%10 != 0 {
		return ErrOrderIncorrectNumber
	}

	return nil
}

var (
	ErrOrderNotExists            error = errors.New("order not exists")
	ErrOrderAlreadyExists        error = errors.New("order already exists")
	ErrOrderBelongsToAnotherUser error = errors.New("order belongs to another user")
	ErrOrderIncorrectNumber      error = errors.New("order incorrect number")
)

type Withdrawal struct {
	OrderNumber string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
	UserID      *int64    `json:"-"`
}

var (
	ErrNoWithdrawals               error = errors.New("no withdrawals by user")
	ErrWithdrawalAlreadyExists     error = errors.New("withdrawal already exists")
	ErrWithdrawalInsufficientFunds error = errors.New("insufficient funds")
)

type OrderAccrual struct {
	Number  string      `json:"order"`
	Status  OrderStatus `json:"status"`
	Accrual float64     `json:"accrual"`
}

func NonFinalStatuses() []OrderStatus {
	return []OrderStatus{OrderStatusNew, OrderStatusProcessing}
}
