//go:build e2e

// Package e2e runs black-box tests against a real HTTP server backed by a real
// MySQL (spun up via testcontainers). Every test drives the public API and then
// asserts both the HTTP response and the actual database state.
//
// Run with:  go test -tags e2e ./e2e/...   (requires Docker)
package e2e

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mysql"

	"github.com/shseooo/go-architecture/internal/bootstrap"
	"github.com/shseooo/go-architecture/migrations"
)

var (
	baseURL string
	testDB  *sql.DB
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	container, err := mysql.Run(ctx, "mysql:8.3",
		mysql.WithDatabase("shop"),
		mysql.WithUsername("user"),
		mysql.WithPassword("password"),
	)
	if err != nil {
		fmt.Println("failed to start mysql container:", err)
		os.Exit(1)
	}

	dsn, err := container.ConnectionString(ctx, "parseTime=true", "loc=UTC")
	if err != nil {
		fmt.Println("failed to build dsn:", err)
		os.Exit(1)
	}
	testDB, err = sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("failed to open db:", err)
		os.Exit(1)
	}

	// apply schema via goose (same migrations as production)
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("mysql"); err != nil {
		fmt.Println("goose dialect:", err)
		os.Exit(1)
	}
	if err := goose.Up(testDB, "."); err != nil {
		fmt.Println("failed to migrate:", err)
		os.Exit(1)
	}

	srv := httptest.NewServer(bootstrap.Handler(testDB, 30*time.Second))
	baseURL = srv.URL

	code := m.Run()

	srv.Close()
	_ = testDB.Close()
	_ = container.Terminate(ctx)
	os.Exit(code)
}

// --- HTTP helpers -----------------------------------------------------------

func do(t *testing.T, method, path string, body any) (int, []byte) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, baseURL+path, reader)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	out, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, out
}

// decode unwraps the standard success envelope {"data": <T>} and returns the data.
func decode[T any](t *testing.T, body []byte) T {
	t.Helper()
	var env struct {
		Data T `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &env), string(body))
	return env.Data
}

// --- tests ------------------------------------------------------------------

func TestOrderLifecycle(t *testing.T) {
	// 1. register a member
	status, body := do(t, http.MethodPost, "/members", map[string]any{
		"name":    "Alice",
		"address": map[string]string{"city": "Seoul", "street": "1", "zipcode": "00001"},
	})
	require.Equal(t, http.StatusCreated, status, string(body))
	member := decode[struct {
		ID int64 `json:"id"`
	}](t, body)
	require.NotZero(t, member.ID)

	// 2. create an item with stock 10
	status, body = do(t, http.MethodPost, "/items", map[string]any{
		"name": "Clean Architecture", "price": 15000, "stock_quantity": 10,
		"type": "BOOK", "author": "Robert C. Martin", "isbn": "9780134494166",
		"category_ids": []int64{2},
	})
	require.Equal(t, http.StatusCreated, status, string(body))
	item := decode[struct {
		ID int64 `json:"id"`
	}](t, body)

	// 3. place an order for 3 units
	status, body = do(t, http.MethodPost, "/orders", map[string]any{
		"member_id": member.ID,
		"address":   map[string]string{"city": "Seoul", "street": "1", "zipcode": "00001"},
		"lines":     []map[string]any{{"item_id": item.ID, "count": 3}},
	})
	require.Equal(t, http.StatusCreated, status, string(body))
	placed := decode[struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}](t, body)
	assert.Equal(t, "ORDER", placed.Status)

	// DB state: stock decremented 10 -> 7
	assert.Equal(t, 7, stockOf(t, item.ID))

	// 4. list the member's orders (N+1-safe path)
	status, body = do(t, http.MethodGet, fmt.Sprintf("/members/%d/orders", member.ID), nil)
	require.Equal(t, http.StatusOK, status)
	orders := decode[[]struct {
		ID         int64 `json:"id"`
		OrderItems []struct {
			ItemName string `json:"item_name"`
			Count    int    `json:"count"`
		} `json:"order_items"`
	}](t, body)
	require.Len(t, orders, 1)
	require.Len(t, orders[0].OrderItems, 1)
	assert.Equal(t, "Clean Architecture", orders[0].OrderItems[0].ItemName)

	// 5. cancel the order
	status, _ = do(t, http.MethodPost, fmt.Sprintf("/orders/%d/cancel", placed.ID), nil)
	require.Equal(t, http.StatusNoContent, status)

	// DB state: order canceled and stock restored 7 -> 10
	assert.Equal(t, "CANCEL", orderStatusOf(t, placed.ID))
	assert.Equal(t, 10, stockOf(t, item.ID))
}

func TestPlaceOrder_InsufficientStock(t *testing.T) {
	memberID := mustCreateMember(t)
	itemID := mustCreateItem(t, "Rare Vinyl", 50000, 1, "ALBUM")

	status, body := do(t, http.MethodPost, "/orders", map[string]any{
		"member_id": memberID,
		"address":   map[string]string{"city": "Seoul"},
		"lines":     []map[string]any{{"item_id": itemID, "count": 5}},
	})
	assert.Equal(t, http.StatusConflict, status, string(body))
	// stock unchanged (transaction rolled back)
	assert.Equal(t, 1, stockOf(t, itemID))
}

func TestCreateItem_InvalidType(t *testing.T) {
	status, _ := do(t, http.MethodPost, "/items", map[string]any{
		"name": "Mystery", "price": 100, "stock_quantity": 1, "type": "GADGET",
	})
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestItemSearch_PriceRangeSortAndPagination(t *testing.T) {
	// distinctive price band so we don't collide with items from other tests
	mustCreateItem(t, "Cheap Movie", 111, 5, "MOVIE")
	mustCreateItem(t, "Pricey Movie", 999, 5, "MOVIE")

	status, body := do(t, http.MethodGet, "/items?minPrice=100&maxPrice=1000&sort=price_asc&limit=1", nil)
	require.Equal(t, http.StatusOK, status)

	var paged struct {
		Data []struct {
			Price int `json:"price"`
		} `json:"data"`
		Meta struct {
			Pagination struct {
				Total   int  `json:"total"`
				Limit   int  `json:"limit"`
				Offset  int  `json:"offset"`
				HasNext bool `json:"has_next"`
			} `json:"pagination"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(body, &paged), string(body))

	// page holds exactly the requested limit
	require.Len(t, paged.Data, 1)
	// pagination metadata reflects the full result set
	assert.GreaterOrEqual(t, paged.Meta.Pagination.Total, 2)
	assert.Equal(t, 1, paged.Meta.Pagination.Limit)
	assert.True(t, paged.Meta.Pagination.HasNext)
}

func TestErrorEnvelope(t *testing.T) {
	status, body := do(t, http.MethodGet, "/members/99999999", nil)
	require.Equal(t, http.StatusNotFound, status)

	var env struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(body, &env), string(body))
	assert.Equal(t, "NOT_FOUND", env.Error.Code)
	assert.NotEmpty(t, env.Error.Message)
}

// --- helpers reading real DB state ------------------------------------------

func stockOf(t *testing.T, itemID int64) int {
	t.Helper()
	var stock int
	require.NoError(t, testDB.QueryRow(`SELECT stock_quantity FROM item WHERE id = ?`, itemID).Scan(&stock))
	return stock
}

func orderStatusOf(t *testing.T, orderID int64) string {
	t.Helper()
	var status string
	require.NoError(t, testDB.QueryRow(`SELECT status FROM orders WHERE id = ?`, orderID).Scan(&status))
	return status
}

func mustCreateMember(t *testing.T) int64 {
	t.Helper()
	status, body := do(t, http.MethodPost, "/members", map[string]any{"name": "Buyer"})
	require.Equal(t, http.StatusCreated, status)
	return decode[struct {
		ID int64 `json:"id"`
	}](t, body).ID
}

func mustCreateItem(t *testing.T, name string, price, stock int, itemType string) int64 {
	t.Helper()
	status, body := do(t, http.MethodPost, "/items", map[string]any{
		"name": name, "price": price, "stock_quantity": stock, "type": itemType,
	})
	require.Equal(t, http.StatusCreated, status, string(body))
	return decode[struct {
		ID int64 `json:"id"`
	}](t, body).ID
}
