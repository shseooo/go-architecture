package rest

import (
	"context"
	"net/http"

	"github.com/shseooo/go-architecture/app/domain"
	"github.com/shseooo/go-architecture/app/service/order"
)

type OrderService interface {
	Place(ctx context.Context, cmd order.PlaceOrderCommand) (domain.Order, error)
	FindByMember(ctx context.Context, memberID int64) ([]domain.Order, error)
	Cancel(ctx context.Context, orderID int64) error
}

type OrderHandler struct {
	svc OrderService
}

func NewOrderHandler(svc OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

type orderLineRequest struct {
	ItemID int64 `json:"item_id"`
	Count  int   `json:"count"`
}

type orderRequest struct {
	MemberID int64              `json:"member_id"`
	Address  domain.Address     `json:"address"`
	Lines    []orderLineRequest `json:"lines"`
}

// Place godoc
// @Summary  상품 주문
// @Description  재고를 확인·차감하고 배송 정보와 함께 주문을 생성한다 (단일 트랜잭션).
// @Tags     orders
// @Accept   json
// @Produce  json
// @Param    body  body      orderRequest  true  "주문 정보"
// @Success  201   {object}  rest.Envelope[domain.Order]
// @Failure  400   {object}  rest.ErrorEnvelope  "입력 검증 실패"
// @Failure  404   {object}  rest.ErrorEnvelope  "회원/상품 없음"
// @Failure  409   {object}  rest.ErrorEnvelope  "재고 부족"
// @Router   /orders [post]
func (h *OrderHandler) Place(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	cmd := order.PlaceOrderCommand{
		MemberID: req.MemberID,
		Address:  req.Address,
		Lines:    make([]order.OrderLine, len(req.Lines)),
	}
	for i, l := range req.Lines {
		cmd.Lines[i] = order.OrderLine{ItemID: l.ItemID, Count: l.Count}
	}
	placed, err := h.svc.Place(r.Context(), cmd)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, placed)
}

// FindByMember godoc
// @Summary  주문 내역 조회
// @Description  회원의 주문 목록을 조회한다. 주문 항목·배송은 배치 로딩으로 N+1 없이 함께 반환된다.
// @Tags     orders
// @Produce  json
// @Param    id   path      int  true  "회원 ID"
// @Success  200  {object}  rest.Envelope[[]domain.Order]
// @Failure  400  {object}  rest.ErrorEnvelope  "잘못된 ID"
// @Router   /members/{id}/orders [get]
func (h *OrderHandler) FindByMember(w http.ResponseWriter, r *http.Request) {
	memberID, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	orders, err := h.svc.FindByMember(r.Context(), memberID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, orders)
}

// Cancel godoc
// @Summary  주문 취소
// @Description  주문을 취소하고 각 상품의 재고를 복원한다 (단일 트랜잭션).
// @Tags     orders
// @Param    id  path  int  true  "주문 ID"
// @Success  204  "취소 완료"
// @Failure  404  {object}  rest.ErrorEnvelope  "주문 없음"
// @Failure  409  {object}  rest.ErrorEnvelope  "이미 취소된 주문"
// @Router   /orders/{id}/cancel [post]
func (h *OrderHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	orderID, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.svc.Cancel(r.Context(), orderID); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
