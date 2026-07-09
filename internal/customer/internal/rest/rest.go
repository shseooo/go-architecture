// Package rest is the customer module's HTTP delivery layer.
package rest

import (
	"context"
	"net/http"

	"github.com/shseooo/go-architecture/internal/customer/internal/domain"
	"github.com/shseooo/go-architecture/internal/platform/httpx"
	"github.com/shseooo/go-architecture/internal/shared"
)

// Service is the use-case contract the handler depends on (consumer-defined).
type Service interface {
	Register(ctx context.Context, m *domain.Member) error
	GetByID(ctx context.Context, id int64) (domain.Member, error)
}

type Handler struct {
	svc Service
}

func New(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /members", h.register)
	mux.HandleFunc("GET /members/{id}", h.getByID)
}

type memberRequest struct {
	Name    string         `json:"name"`
	Address shared.Address `json:"address"`
}

// MemberResponse is the customer resource returned to clients.
type MemberResponse struct {
	ID      int64          `json:"id"`
	Name    string         `json:"name"`
	Address shared.Address `json:"address"`
}

func toMemberResponse(m domain.Member) MemberResponse {
	return MemberResponse{ID: m.ID, Name: m.Name, Address: m.Address}
}

// register godoc
// @Summary  회원 등록
// @Tags     members
// @Accept   json
// @Param    body  body  memberRequest  true  "회원 정보"
// @Success  201
// @Failure  400
// @Router   /members [post]
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req memberRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	m := &domain.Member{Name: req.Name, Address: req.Address}
	if err := h.svc.Register(r.Context(), m); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Data(w, http.StatusCreated, toMemberResponse(*m))
}

// getByID godoc
// @Summary  회원 조회
// @Tags     members
// @Param    id   path  int  true  "회원 ID"
// @Success  200
// @Failure  404
// @Router   /members/{id} [get]
func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	m, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Data(w, http.StatusOK, toMemberResponse(m))
}
