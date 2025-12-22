package accrual

import (
	"context"
	"log"
	"math"
	"strconv"
	"time"
)

type Repo interface {
	ListPendingOrders(ctx context.Context, limit int) ([]int64, error)
	ApplyOrderProcessedOnce(ctx context.Context, number int64, accural int64) error
	MarkOrderInvalidOnce(ctx context.Context, number int64) error
	UpdateOrderStatusNonFinal(ctx context.Context, number int64, status string) error
}

type AccrualClient interface {
	GetOrder(ctx context.Context, number string) (OrderInfo, error)
}

type AccrualWorker struct {
	repo       Repo
	client     AccrualClient
	logger     *log.Logger
	pollEvery  time.Duration
	batchLimit int
	reqTimeout time.Duration
}

func NewAccrualWorker(repo Repo, client AccrualClient, logger *log.Logger) *AccrualWorker {
	if logger == nil {
		logger = log.Default()
	}
	return &AccrualWorker{
		repo:   repo,
		client: client,
		logger: logger,

		pollEvery:  50 * time.Millisecond,
		batchLimit: 100,
		reqTimeout: 300 * time.Millisecond,
	}
}

func (w *AccrualWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.pollEvery)
	defer ticker.Stop()

	var pauseUntil time.Time

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			if !pauseUntil.IsZero() && time.Now().Before(pauseUntil) {
				continue
			}

			numbers, err := w.repo.ListPendingOrders(ctx, w.batchLimit)
			if err != nil {
				w.logger.Printf("accrual worker: ListPendingOrders: %v", err)
				continue
			}
			if len(numbers) == 0 {
				continue
			}

			for _, num := range numbers {
				reqCtx, cancel := context.WithTimeout(ctx, w.reqTimeout)
				info, err := w.client.GetOrder(reqCtx, itoa64(num))
				cancel()

				if err != nil {
					if err == ErrNotRegistered {
						continue
					}
					if rl, ok := err.(RateLimitError); ok {
						pauseUntil = time.Now().Add(rl.RetryAfter)
						w.logger.Printf("accrual worker: 429 rate limited, pause %s", rl.RetryAfter)
						break
					}
					w.logger.Printf("accrual worker: GetOrder %d: %v", num, err)
					continue
				}

				switch info.Status {
				case StatusProcessed:
					acc := 0.0
					if info.Accrual != nil {
						acc = *info.Accrual
					}
					accural := moneyToCents(acc)

					if err := w.repo.ApplyOrderProcessedOnce(ctx, num, accural); err != nil {
						w.logger.Printf("accrual worker: ApplyOrderProcessedOnce %d: %v", num, err)
					}

				case StatusInvalid:
					if err := w.repo.MarkOrderInvalidOnce(ctx, num); err != nil {
						w.logger.Printf("accrual worker: MarkOrderInvalidOnce %d: %v", num, err)
					}

				case StatusRegistered, StatusProcessing:
					if err := w.repo.UpdateOrderStatusNonFinal(ctx, num, string(info.Status)); err != nil {
						w.logger.Printf("accrual worker: UpdateOrderStatusNonFinal %d: %v", num, err)
					}

				default:
					if err := w.repo.UpdateOrderStatusNonFinal(ctx, num, string(info.Status)); err != nil {
						w.logger.Printf("accrual worker: UpdateOrderStatusNonFinal %d: %v", num, err)
					}
				}
			}
		}
	}
}

func moneyToCents(v float64) int64 {
	return int64(math.Round(v * 100))
}

func itoa64(v int64) string {
	return strconvFormatInt(v)
}

func strconvFormatInt(v int64) string {
	return strconv.FormatInt(v, 10)
}
