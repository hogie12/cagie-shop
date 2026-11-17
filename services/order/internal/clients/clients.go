// Package clients calls the catalog service over HTTP. Reads only: anything
// that changes another service's data goes through events instead.
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	ErrNotFound = errors.New("not found")
)

type Product struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	PriceIDR int64  `json:"priceIdr"`
}

// Catalog is the base URL of the catalog service, e.g. http://catalog:8081.
type Catalog struct{ BaseURL string }

// otelhttp passes the trace on to the catalog service.
var httpClient = &http.Client{Timeout: 3 * time.Second, Transport: otelhttp.NewTransport(http.DefaultTransport)}

func (c Catalog) GetProduct(ctx context.Context, id int64) (Product, error) {
	var p Product
	status, err := call(ctx, http.MethodGet, c.BaseURL+"/products/"+strconv.FormatInt(id, 10), nil, &p)
	if err != nil {
		return Product{}, err
	}
	switch status {
	case http.StatusOK:
		return p, nil
	case http.StatusNotFound:
		return Product{}, ErrNotFound
	}
	return Product{}, fmt.Errorf("catalog returned %d", status)
}

func call(ctx context.Context, method, url string, body, out any) (int, error) {
	var rd io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rd)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if out != nil && resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(out); err != nil {
			return 0, err
		}
	}
	return resp.StatusCode, nil
}
