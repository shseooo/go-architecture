// Package rest is the catalog module's HTTP delivery layer.
package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/shseooo/go-architecture/internal/catalog/internal/domain"
	"github.com/shseooo/go-architecture/internal/platform/httpx"
)

// Service is the use-case contract the handler depends on.
type Service interface {
	Create(ctx context.Context, it *domain.Item) error
	Update(ctx context.Context, it *domain.Item) error
	GetByID(ctx context.Context, id int64) (domain.Item, error)
	Search(ctx context.Context, f domain.ItemFilter) ([]domain.Item, int, error)
}

type Handler struct {
	svc Service
}

func New(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /items", h.create)
	mux.HandleFunc("GET /items", h.search)
	mux.HandleFunc("GET /items/{id}", h.getByID)
	mux.HandleFunc("PUT /items/{id}", h.update)
}

type itemRequest struct {
	Name          string          `json:"name"`
	Price         int             `json:"price"`
	StockQuantity int             `json:"stock_quantity"`
	Type          domain.ItemType `json:"type"`
	Author        string          `json:"author,omitempty"`
	ISBN          string          `json:"isbn,omitempty"`
	Artist        string          `json:"artist,omitempty"`
	Etc           string          `json:"etc,omitempty"`
	Director      string          `json:"director,omitempty"`
	Actor         string          `json:"actor,omitempty"`
	CategoryIDs   []int64         `json:"category_ids,omitempty"`
}

func (req itemRequest) toDomain() *domain.Item {
	return &domain.Item{
		Name: req.Name, Price: req.Price, StockQuantity: req.StockQuantity, Type: req.Type,
		Author: req.Author, ISBN: req.ISBN, Artist: req.Artist, Etc: req.Etc,
		Director: req.Director, Actor: req.Actor, CategoryIDs: req.CategoryIDs,
	}
}

// ItemResponse is the item resource returned to clients.
type ItemResponse struct {
	ID            int64           `json:"id"`
	Name          string          `json:"name"`
	Price         int             `json:"price"`
	StockQuantity int             `json:"stock_quantity"`
	Type          domain.ItemType `json:"type"`
	CreatedAt     time.Time       `json:"created_at"`
	Author        string          `json:"author,omitempty"`
	ISBN          string          `json:"isbn,omitempty"`
	Artist        string          `json:"artist,omitempty"`
	Etc           string          `json:"etc,omitempty"`
	Director      string          `json:"director,omitempty"`
	Actor         string          `json:"actor,omitempty"`
	CategoryIDs   []int64         `json:"category_ids,omitempty"`
}

func toItemResponse(it domain.Item) ItemResponse {
	return ItemResponse{
		ID: it.ID, Name: it.Name, Price: it.Price, StockQuantity: it.StockQuantity,
		Type: it.Type, CreatedAt: it.CreatedAt,
		Author: it.Author, ISBN: it.ISBN, Artist: it.Artist, Etc: it.Etc,
		Director: it.Director, Actor: it.Actor, CategoryIDs: it.CategoryIDs,
	}
}

// create godoc
// @Summary  상품 등록
// @Tags     items
// @Accept   json
// @Param    body  body  itemRequest  true  "상품 정보 (type: BOOK|ALBUM|MOVIE)"
// @Success  201
// @Failure  400
// @Router   /items [post]
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req itemRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	it := req.toDomain()
	if err := h.svc.Create(r.Context(), it); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Data(w, http.StatusCreated, toItemResponse(*it))
}

// update godoc
// @Summary  상품 수정
// @Tags     items
// @Param    id    path  int          true  "상품 ID"
// @Param    body  body  itemRequest  true  "상품 정보"
// @Success  200
// @Failure  404
// @Router   /items/{id} [put]
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	var req itemRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	it := req.toDomain()
	it.ID = id
	if err := h.svc.Update(r.Context(), it); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Data(w, http.StatusOK, toItemResponse(*it))
}

// getByID godoc
// @Summary  상품 조회
// @Tags     items
// @Param    id  path  int  true  "상품 ID"
// @Success  200
// @Failure  404
// @Router   /items/{id} [get]
func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.PathID(r, "id")
	if err != nil {
		httpx.Error(w, err)
		return
	}
	it, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.Data(w, http.StatusOK, toItemResponse(it))
}

// search godoc
// @Summary  상품 검색 (동적 쿼리 + 페이징)
// @Tags     items
// @Param    categoryId  query  int     false  "카테고리 ID"
// @Param    minPrice    query  int     false  "최소 가격"
// @Param    maxPrice    query  int     false  "최대 가격"
// @Param    sort        query  string  false  "정렬"  Enums(newest, oldest, price_asc, price_desc)
// @Param    limit       query  int     false  "페이지 크기 (기본 20, 최대 100)"
// @Param    offset      query  int     false  "오프셋"
// @Success  200
// @Router   /items [get]
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	f := domain.ItemFilter{
		CategoryID: httpx.QueryInt64(r, "categoryId"),
		MinPrice:   httpx.QueryInt(r, "minPrice"),
		MaxPrice:   httpx.QueryInt(r, "maxPrice"),
		Sort:       domain.ItemSort(r.URL.Query().Get("sort")),
	}
	if v := httpx.QueryInt(r, "limit"); v != nil {
		f.Limit = *v
	}
	if v := httpx.QueryInt(r, "offset"); v != nil {
		f.Offset = *v
	}
	f.Normalize()

	items, total, err := h.svc.Search(r.Context(), f)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	out := make([]ItemResponse, len(items))
	for i, it := range items {
		out[i] = toItemResponse(it)
	}
	httpx.Paged(w, out, httpx.Pagination{
		Total: total, Limit: f.Limit, Offset: f.Offset,
		HasNext: f.Offset+len(items) < total,
	})
}
