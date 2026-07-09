// Package rest is the ordering module's HTTP delivery layer.
package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/shseooo/go-architecture/internal/ordering/internal/domain"
	"github.com/shseooo/go-architecture/internal/platform/httpx"
	"github.com/shseooo/go-architecture/internal/shared"
)

// Service is the use-case contract the handler depends on.
type Service interface {
	Place(ctx context.Context, cmd domain.PlaceCommand) (domain.Order, error)
	FindByMember(ctx context.Context, memberID int64) ([]domain.Order, error)
	Cancel(ctx context.Context, orderID int64) error
}

type Handler struct {
	svc Service
}

func New(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /orders", h.place)
	mux.HandleFunc("GET /members/{id}/orders", h.findByMember)
	mux.HandleFunc("POST /orders/{id}/cancel", h.cancel)
}

type orderRequest struct {
	MemberID int64          `json:"member_id"`
	Address  shared.Address `json:"address"`
	Lines    []struct {
		ItemID int64 `json:"item_id"`
		Count  int   `json:"count"`
	} `json:"lines"`
}

// OrderResponse is the order resource returned to clients.
type OrderResponse struct {
	ID         int64               `json:"id"`
	MemberID   int64               `json:"member_id"`
	Status     domain.OrderStatus  `json:"status"`
	OrderDate  time.Time           `json:"order_date"`
	TotalPrice int                 `json:"total_price"`
	OrderItems []orderItemResponse `json:"order_items"`
	Delivery   *deliveryResponse   `json:"delivery,omitempty"`
}

type orderItemResponse struct {
	ItemID     int64  `json:"item_id"`
	ItemName   string `json:"item_name"`
	OrderPrice int    `json:"order_price"`
	Count      int    `json:"count"`
}

type deliveryResponse struct {
	Status  domain.DeliveryStatus `json:"status"`
	Address shared.Address        `json:"address"`
}

func toOrderResponse(o domain.Order) OrderResponse {
	resp := OrderResponse{
		ID: o.ID, MemberID: o.MemberID, Status: o.Status,
		OrderDate: o.OrderDate, TotalPrice: o.TotalPrice(),
		OrderItems: make([]orderItemResponse, len(o.OrderItems)),
	}
	for i, oi := range o.OrderItems {
		resp.OrderItems[i] = orderItemResponse{
			ItemID: oi.ItemID, ItemName: oi.ItemName, OrderPrice: oi.OrderPrice, Count: oi.Count,
		}
	}
	if o.Delivery != nil {
		resp.Delivery = &deliveryResponse{Status: o.Delivery.Status, Address: o.Delivery.Address}
	}
	return resp
}

// place godoc
// @Summary  상품 주문
// @Tags     orders
// @Accept   json
// @Param    body  body  orderRequest  true  "주문 정보"
// @Success  201
// @Failure  400
// @Failure  404
// @Failure  409
// @Router   /orders [post]
func (h *Handler) place(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	cmd := domain.PlaceCommand{MemberID: req.MemberID, Address: req.Address}
	for _, l := range req.Lines {
		cmd.Lines = append(cmd.Lines, domain.Line{ItemID: l.ItemID, Count: l.Count})
	}
	placed, err := h.svc.Place(r.Context(), cmd)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Data(w, http.StatusCreated, toOrderResponse(placed))
}

// findByMember godoc
// @Summary  주문 내역 조회
// @Tags     orders
// @Param    id  path  int  true  "회원 ID"
// @Success  200
// @Router   /members/{id}/orders [get]
func (h *Handler) findByMember(w http.ResponseWriter, r *http.Request) {
	memberID, err := httpx.PathID(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	orders, err := h.svc.FindByMember(r.Context(), memberID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	out := make([]OrderResponse, len(orders))
	for i, o := range orders {
		out[i] = toOrderResponse(o)
	}
	httpx.Data(w, http.StatusOK, out)
}

// cancel godoc
// @Summary  주문 취소
// @Tags     orders
// @Param    id  path  int  true  "주문 ID"
// @Success  204
// @Failure  404
// @Failure  409
// @Router   /orders/{id}/cancel [post]
func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	orderID, err := httpx.PathID(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.Cancel(r.Context(), orderID); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.NoContent(w)
}
