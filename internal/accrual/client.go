package accrual

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"github.com/g123udini/gofemart/internal/service"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

var ErrOrderNotFound = errors.New("order not found")
var ErrInternalServerError = errors.New("internal server error")

type ErrUnexpectedStatus struct {
	Status int
}

type Client struct {
	gzipPool *sync.Pool
	resty    *resty.Client
}

func New(address string) *Client {
	client := &Client{
		resty: resty.
			New().
			SetTransport(http.DefaultTransport).
			SetBaseURL(address).
			SetHeader("Accept-Encoding", "gzip"),
	}

	return client
}

func (client *Client) GetAccrual(ctx context.Context, orderID uint64) (*Accrual, error) {
	result, err := client.doRequest(client.createRequest(ctx).
		SetPathParams(map[string]string{
			"orderID": strconv.FormatUint(orderID, 10),
		}).
		SetResult(&Accrual{}),
		resty.MethodGet, "api/orders/{orderID}")

	if err != nil {
		return nil, err
	}

	return result.Result().(*Accrual), nil
}

func (client *Client) createRequest(ctx context.Context) *resty.Request {
	return client.resty.R().SetContext(ctx)
}

func (client *Client) doRequest(request *resty.Request, method, url string) (*resty.Response, error) {
	var result *resty.Response
	do := func() error {
		var err error
		result, err = request.Execute(method, url)
		if err != nil {
			return err
		}

		switch result.StatusCode() {
		case http.StatusOK:
			return nil
		case http.StatusNoContent:
			return ErrOrderNotFound
		case http.StatusTooManyRequests:
			return newErrTooManyRequests(service.Parse(result.Header().Get("Retry-After"), time.Second*10))
		case http.StatusInternalServerError:
			return ErrInternalServerError
		default:
			return newErrUnexpectedStatus(result.StatusCode())
		}
	}

	err := service.Retry(time.Second, 5*time.Second, 4, 2, do, func(err error) bool {
		return !errors.As(err, &ErrUnexpectedStatus{}) &&
			!errors.As(err, &ErrTooManyRequests{}) &&
			!errors.Is(err, context.DeadlineExceeded) &&
			!errors.Is(err, context.Canceled)
	})

	return result, err
}

func (client *Client) compressRequestBody(resty *resty.Client, request *http.Request) error {
	if request.Body == nil {
		return nil
	}

	buffer := bytes.NewBuffer([]byte{})
	writer := client.gzipPool.Get().(*gzip.Writer)
	defer client.gzipPool.Put(writer)
	writer.Reset(buffer)

	_, err := io.Copy(writer, request.Body)
	if err = errors.Join(err, writer.Close(), request.Body.Close()); err != nil {
		return err
	}

	request.Body = io.NopCloser(buffer)
	request.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buffer.Bytes())), nil
	}
	request.ContentLength = int64(buffer.Len())
	request.Header.Set("Content-Encoding", "gzip")
	request.Header.Set("Content-Length", fmt.Sprintf("%d", buffer.Len()))

	return nil
}

func (err ErrUnexpectedStatus) Error() string {
	return fmt.Sprintf("unexpected status code: %d", err.Status)
}

func newErrUnexpectedStatus(status int) ErrUnexpectedStatus {
	return ErrUnexpectedStatus{
		Status: status,
	}
}

type ErrTooManyRequests struct {
	RetryAfterTime time.Time
}

func (err ErrTooManyRequests) Error() string {
	return "too many requests"
}

func newErrTooManyRequests(retryAfter time.Duration) ErrTooManyRequests {
	return ErrTooManyRequests{
		RetryAfterTime: time.Now().Add(retryAfter),
	}
}
