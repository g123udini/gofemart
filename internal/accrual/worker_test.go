package accrual

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"
	"testing"
	"time"
)

type fakeRepo struct {
	mu sync.Mutex

	// поведение ListPendingOrders
	pendingBatches [][]int64
	listCalls      int

	// записи вызовов
	processed []struct {
		num     int64
		accural int64
	}
	invalid []int64
	updated []struct {
		num    int64
		status string
	}

	// каналы для синхронизации тестов
	onProcessed chan struct{}
	onInvalid   chan struct{}
	onUpdated   chan struct{}
}

func (r *fakeRepo) ListPendingOrders(ctx context.Context, limit int) ([]int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.listCalls++
	if len(r.pendingBatches) == 0 {
		return nil, nil
	}
	b := r.pendingBatches[0]
	r.pendingBatches = r.pendingBatches[1:]
	return b, nil
}

func (r *fakeRepo) ApplyOrderProcessedOnce(ctx context.Context, number int64, accural int64) error {
	r.mu.Lock()
	r.processed = append(r.processed, struct {
		num     int64
		accural int64
	}{number, accural})
	ch := r.onProcessed
	r.mu.Unlock()

	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	return nil
}

func (r *fakeRepo) MarkOrderInvalidOnce(ctx context.Context, number int64) error {
	r.mu.Lock()
	r.invalid = append(r.invalid, number)
	ch := r.onInvalid
	r.mu.Unlock()

	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	return nil
}

func (r *fakeRepo) UpdateOrderStatusNonFinal(ctx context.Context, number int64, status string) error {
	r.mu.Lock()
	r.updated = append(r.updated, struct {
		num    int64
		status string
	}{number, status})
	ch := r.onUpdated
	r.mu.Unlock()

	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	return nil
}

type fakeClient struct {
	mu sync.Mutex

	// на каждый вызов GetOrder отдаём следующий результат
	results []getOrderResult
	calls   []string
}

type getOrderResult struct {
	info OrderInfo
	err  error
}

func (c *fakeClient) GetOrder(ctx context.Context, number string) (OrderInfo, error) {
	c.mu.Lock()
	c.calls = append(c.calls, number)
	if len(c.results) == 0 {
		c.mu.Unlock()
		return OrderInfo{}, errors.New("no stubbed result")
	}
	r := c.results[0]
	c.results = c.results[1:]
	c.mu.Unlock()
	return r.info, r.err
}

func TestAccrualWorker_Processed_UpdatesOrderAndBalanceOnce(t *testing.T) {
	repo := &fakeRepo{
		pendingBatches: [][]int64{{101}},
		onProcessed:    make(chan struct{}, 1),
	}
	acc := 7.2998 // -> 730 (если округление), но мы хотим 729.98 => 72998, поэтому используем 729.98
	acc = 729.98

	client := &fakeClient{
		results: []getOrderResult{{
			info: OrderInfo{
				Order:   "101",
				Status:  StatusProcessed,
				Accrual: &acc,
			},
		}},
	}

	w := NewAccrualWorker(repo, client, log.New(os.Stdout, "", 0))
	w.pollEvery = 5 * time.Millisecond
	w.reqTimeout = 50 * time.Millisecond
	w.batchLimit = 10

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Run(ctx)

	select {
	case <-repo.onProcessed:
		cancel()
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting processed")
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if len(repo.processed) != 1 {
		t.Fatalf("expected 1 processed call, got %d", len(repo.processed))
	}
	if repo.processed[0].num != 101 {
		t.Fatalf("number mismatch: %d", repo.processed[0].num)
	}
	if repo.processed[0].accural != 72998 {
		t.Fatalf("expected accural 72998, got %d", repo.processed[0].accural)
	}
}

func TestAccrualWorker_Invalid_MarksInvalid(t *testing.T) {
	repo := &fakeRepo{
		pendingBatches: [][]int64{{202}},
		onInvalid:      make(chan struct{}, 1),
	}
	client := &fakeClient{
		results: []getOrderResult{{
			info: OrderInfo{Order: "202", Status: StatusInvalid},
		}},
	}

	w := NewAccrualWorker(repo, client, log.New(os.Stdout, "", 0))
	w.pollEvery = 5 * time.Millisecond
	w.reqTimeout = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Run(ctx)

	select {
	case <-repo.onInvalid:
		cancel()
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting invalid")
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if len(repo.invalid) != 1 || repo.invalid[0] != 202 {
		t.Fatalf("invalid calls mismatch: %+v", repo.invalid)
	}
}

func TestAccrualWorker_Processing_UpdatesStatus(t *testing.T) {
	repo := &fakeRepo{
		pendingBatches: [][]int64{{303}},
		onUpdated:      make(chan struct{}, 1),
	}
	client := &fakeClient{
		results: []getOrderResult{{
			info: OrderInfo{Order: "303", Status: StatusProcessing},
		}},
	}

	w := NewAccrualWorker(repo, client, log.New(os.Stdout, "", 0))
	w.pollEvery = 5 * time.Millisecond
	w.reqTimeout = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go w.Run(ctx)

	select {
	case <-repo.onUpdated:
		cancel()
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("timeout waiting updated")
	}

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if len(repo.updated) != 1 {
		t.Fatalf("expected 1 update, got %d", len(repo.updated))
	}
	if repo.updated[0].num != 303 || repo.updated[0].status != string(StatusProcessing) {
		t.Fatalf("update mismatch: %+v", repo.updated[0])
	}
}

func TestAccrualWorker_NotRegistered_JustSkips(t *testing.T) {
	repo := &fakeRepo{
		pendingBatches: [][]int64{{404}},
	}
	client := &fakeClient{
		results: []getOrderResult{{
			err: ErrNotRegistered,
		}},
	}

	w := NewAccrualWorker(repo, client, log.New(os.Stdout, "", 0))
	w.pollEvery = 5 * time.Millisecond
	w.reqTimeout = 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	w.Run(ctx)

	repo.mu.Lock()
	defer repo.mu.Unlock()

	if len(repo.processed) != 0 || len(repo.invalid) != 0 || len(repo.updated) != 0 {
		t.Fatalf("expected no repo updates, got processed=%d invalid=%d updated=%d",
			len(repo.processed), len(repo.invalid), len(repo.updated))
	}
}

func TestAccrualWorker_RateLimit_Pauses(t *testing.T) {
	repo := &fakeRepo{
		// на первом тике отдаём заказ, дальше ещё раз отдаём — но worker должен “паузы” выдержать
		pendingBatches: [][]int64{{1}, {2}, {3}},
	}
	client := &fakeClient{
		results: []getOrderResult{
			{err: RateLimitError{RetryAfter: 200 * time.Millisecond}},
			{info: OrderInfo{Order: "2", Status: StatusInvalid}}, // если бы не пауза, дошли бы
			{info: OrderInfo{Order: "3", Status: StatusInvalid}},
		},
	}

	w := NewAccrualWorker(repo, client, log.New(os.Stdout, "", 0))
	w.pollEvery = 5 * time.Millisecond
	w.reqTimeout = 50 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()

	w.Run(ctx)

	// за 80мс при RetryAfter=200мс worker должен сделать только 1 запрос в accrual
	client.mu.Lock()
	defer client.mu.Unlock()

	if len(client.calls) != 1 {
		t.Fatalf("expected 1 GetOrder call due to pause, got %d calls: %+v", len(client.calls), client.calls)
	}
}

func TestMoneyToCents(t *testing.T) {
	if got := moneyToCents(729.98); got != 72998 {
		t.Fatalf("got %d want 72998", got)
	}
	if got := moneyToCents(0); got != 0 {
		t.Fatalf("got %d want 0", got)
	}
}

func TestItoa64(t *testing.T) {
	if got := itoa64(123); got != "123" {
		t.Fatalf("got %q want %q", got, "123")
	}
}
