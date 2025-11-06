package accrual

import (
	"errors"
	"syscall"

	"github.com/galogen13/gophermart-loyalty/internal/classification"
	"github.com/galogen13/gophermart-loyalty/internal/retry"
)

type AccrualServiceErrorClassifier struct{}

func NewAccrualServiceErrorClassifier() *AccrualServiceErrorClassifier {
	return &AccrualServiceErrorClassifier{}
}

func (c *AccrualServiceErrorClassifier) Classify(err error) retry.ErrorClassification {
	if err == nil {
		return retry.NonRetriable
	}

	var reqErr syscall.Errno
	if errors.As(err, &reqErr) {
		return classifySyscallError(reqErr)
	}

	return retry.NonRetriable
}

func classifySyscallError(reqErr syscall.Errno) retry.ErrorClassification {

	if classification.IsRetriableSyscallError(reqErr) {
		return retry.Retriable
	}

	return retry.NonRetriable
}
