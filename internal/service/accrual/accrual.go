package accrual

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/galogen13/gophermart-loyalty/internal/logger"
	"github.com/galogen13/gophermart-loyalty/internal/retry"
	"github.com/galogen13/gophermart-loyalty/internal/service/market"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type AccrualService struct {
	workersCount int
	jobs         chan Job
	accruals     chan market.OrderAccrual
	client       *resty.Client
	host         string
	pathSeq      []string
	pauseCond    *sync.Cond
	isPaused     bool
}

func NewAccrualService(host string, workersCount int) *AccrualService {
	return &AccrualService{
		workersCount: workersCount,
		jobs:         make(chan Job, workersCount*2),
		accruals:     make(chan market.OrderAccrual, workersCount*2),
		client:       resty.New(),
		host:         host,
		pathSeq:      []string{"api", "orders"},
		pauseCond:    sync.NewCond(&sync.Mutex{})}
}

type Job struct {
	OrderNumber string
}

type Result struct {
	err          error
	needPause    bool
	retryAfter   int
	orderAccrual OrderAccrual
}

type OrderAccrual struct {
	Number  string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual"`
}

var (
	ErrOrderNotFound = errors.New("order not found in accrual system")
)

func (as *AccrualService) Start(ctx context.Context) {
	for i := 0; i < as.workersCount; i++ {
		go as.worker(ctx)
	}
}

func (as *AccrualService) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-as.jobs:
			if !ok {
				return
			}

			as.checkPause()

			result := as.getOrderAccrual(ctx, job)

			if result.err != nil {
				if errors.Is(result.err, ErrOrderNotFound) {
					logger.Log.Info("error getting order accruals", zap.Error(result.err), zap.String("order number", job.OrderNumber))
				} else {
					logger.Log.Error("error getting order accruals", zap.Error(result.err), zap.String("order number", job.OrderNumber))
				}
				continue
			}

			if result.needPause {
				as.activatePause(time.Duration(result.retryAfter) * time.Second)
				continue
			}

			orderAAccrual := market.OrderAccrual{}
			orderAAccrual.Number = result.orderAccrual.Number
			orderAAccrual.Accrual = result.orderAccrual.Accrual
			status, err := convertAccrualStatusToOrderStatus(result.orderAccrual.Status)
			if err != nil {
				logger.Log.Error("error getting order accruals", zap.Error(result.err), zap.String("order number", job.OrderNumber))
				continue
			}
			orderAAccrual.Status = status

			as.accruals <- orderAAccrual
		}
	}
}

func convertAccrualStatusToOrderStatus(status string) (market.OrderStatus, error) {
	switch status {
	case "REGISTERED":
		return market.OrderStatusNew, nil
	case "INVALID":
		return market.OrderStatusInvalid, nil
	case "PROCESSING":
		return market.OrderStatusProcessing, nil
	case "PROCESSED":
		return market.OrderStatusProcessed, nil
	}
	return "", errors.New("unexpected status from accrual service")
}

func (as *AccrualService) getOrderAccrual(ctx context.Context, job Job) Result {

	// client := resty.New()
	// client.SetRedirectPolicy(resty.RedirectPolicyFunc(
	// 	func(req *http.Request, _ []*http.Request) error {
	// 		req.Method = http.MethodPost
	// 		return nil
	// 	}))

	result := Result{}

	baseURL, err := url.Parse(as.host)
	if err != nil {
		result.err = err
		return result
	}

	as.pathSeq = append(as.pathSeq, job.OrderNumber)
	baseURL = baseURL.JoinPath(as.pathSeq...)

	// baseURL := &url.URL{
	// 	Scheme: "http",
	// 	Host:   as.host,
	// 	Path:   path,
	// }
	fullURL := baseURL.String()

	resp, err := retry.DoWithResult(
		ctx,
		func() (*resty.Response, error) {
			return as.client.R().Get(fullURL)
		},
		NewAccrualServiceErrorClassifier())

	if err != nil {
		result.err = err
		return result
	}

	if resp.StatusCode() == http.StatusTooManyRequests {
		retryAfterStr := resp.Header().Get("Retry-After")
		if retryAfterStr != "" {
			retryAfter, err := strconv.Atoi(retryAfterStr)
			result.retryAfter = retryAfter
			result.err = err
			result.needPause = (err == nil)
		} else {
			result.err = errors.New("no header Retry-After in responce or header is empty")
		}
		return result
	}

	if resp.StatusCode() == http.StatusNoContent {
		result.err = ErrOrderNotFound
		return result
	}

	if resp.StatusCode() != http.StatusOK {
		result.err = fmt.Errorf("unexpected status code: %d", resp.StatusCode())
		return result
	}

	bodyBytes := resp.Body()

	ordAcc := &OrderAccrual{}
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(ordAcc); err != nil {
		result.err = err
		return result
	}

	result.orderAccrual = *ordAcc
	return result
}

func (as *AccrualService) checkPause() {
	as.pauseCond.L.Lock()
	defer as.pauseCond.L.Unlock()

	for as.isPaused {
		as.pauseCond.Wait()
	}
}

func (as *AccrualService) activatePause(duration time.Duration) {
	as.pauseCond.L.Lock()
	defer as.pauseCond.L.Unlock()

	if !as.isPaused {
		as.isPaused = true
		logger.Log.Info("accrual system pause activated", zap.Duration("duration", duration))

		time.AfterFunc(duration, func() {
			as.deactivatePause()
		})
	}
}

func (as *AccrualService) deactivatePause() {
	as.pauseCond.L.Lock()
	as.isPaused = false
	logger.Log.Info("accrual system pause deactivated")
	as.pauseCond.Broadcast()
	as.pauseCond.L.Unlock()
}

func (as *AccrualService) AddJob(job Job) {
	as.jobs <- job
}

func (as *AccrualService) GetResults() <-chan market.OrderAccrual {
	return as.accruals
}

func (as *AccrualService) Close() {
	close(as.jobs)
	close(as.accruals)
}
