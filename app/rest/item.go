package rest

import (
	"context"
	"net/http"

	"github.com/shseooo/go-architecture/app/domain"
)

type ItemService interface {
	Create(ctx context.Context, it *domain.Item) error
	Update(ctx context.Context, it *domain.Item) error
	GetByID(ctx context.Context, id int64) (domain.Item, error)
	Search(ctx context.Context, f domain.ItemFilter) ([]domain.Item, int, error)
}

type ItemHandler struct {
	svc ItemService
}

func NewItemHandler(svc ItemService) *ItemHandler {
	return &ItemHandler{svc: svc}
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
		Name:          req.Name,
		Price:         req.Price,
		StockQuantity: req.StockQuantity,
		Type:          req.Type,
		Author:        req.Author,
		ISBN:          req.ISBN,
		Artist:        req.Artist,
		Etc:           req.Etc,
		Director:      req.Director,
		Actor:         req.Actor,
		CategoryIDs:   req.CategoryIDs,
	}
}

// Create godoc
// @Summary  상품 등록
// @Tags     items
// @Accept   json
// @Produce  json
// @Param    body  body      itemRequest  true  "상품 정보 (type: BOOK|ALBUM|MOVIE)"
// @Success  201   {object}  rest.Envelope[domain.Item]
// @Failure  400   {object}  rest.ErrorEnvelope  "입력 검증 실패"
// @Failure  500   {object}  rest.ErrorEnvelope  "서버 에러"
// @Router   /items [post]
func (h *ItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req itemRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	it := req.toDomain()
	if err := h.svc.Create(r.Context(), it); err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, it)
}

// Update godoc
// @Summary  상품 수정
// @Tags     items
// @Accept   json
// @Produce  json
// @Param    id    path      int          true  "상품 ID"
// @Param    body  body      itemRequest  true  "상품 정보"
// @Success  200   {object}  rest.Envelope[domain.Item]
// @Failure  400   {object}  rest.ErrorEnvelope  "입력 검증 실패"
// @Failure  404   {object}  rest.ErrorEnvelope  "상품 없음"
// @Router   /items/{id} [put]
func (h *ItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req itemRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	it := req.toDomain()
	it.ID = id
	if err := h.svc.Update(r.Context(), it); err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, it)
}

// GetByID godoc
// @Summary  상품 조회
// @Tags     items
// @Produce  json
// @Param    id   path      int  true  "상품 ID"
// @Success  200  {object}  rest.Envelope[domain.Item]
// @Failure  404  {object}  rest.ErrorEnvelope  "상품 없음"
// @Router   /items/{id} [get]
func (h *ItemHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	it, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, it)
}

// Search godoc
// @Summary  상품 검색 (동적 쿼리)
// @Description  카테고리·가격범위 필터와 날짜/가격 정렬을 지원한다. 정렬 키는 화이트리스트로 제한된다.
// @Tags     items
// @Produce  json
// @Param    categoryId  query  int     false  "카테고리 ID"
// @Param    minPrice    query  int     false  "최소 가격"
// @Param    maxPrice    query  int     false  "최대 가격"
// @Param    sort        query  string  false  "정렬"  Enums(newest, oldest, price_asc, price_desc)
// @Param    limit       query  int     false  "페이지 크기 (기본 20, 최대 100)"
// @Param    offset      query  int     false  "오프셋 (기본 0)"
// @Success  200  {object}  rest.Envelope[[]domain.Item]
// @Router   /items [get]
func (h *ItemHandler) Search(w http.ResponseWriter, r *http.Request) {
	f := domain.ItemFilter{
		CategoryID: queryInt64(r, "categoryId"),
		MinPrice:   queryInt(r, "minPrice"),
		MaxPrice:   queryInt(r, "maxPrice"),
		Sort:       domain.ItemSort(r.URL.Query().Get("sort")),
	}
	if v := queryInt(r, "limit"); v != nil {
		f.Limit = *v
	}
	if v := queryInt(r, "offset"); v != nil {
		f.Offset = *v
	}
	f.Normalize() // so the response pagination reflects the effective window

	items, total, err := h.svc.Search(r.Context(), f)
	if err != nil {
		writeError(w, err)
		return
	}

	writePaged(w, items, Pagination{
		Total:   total,
		Limit:   f.Limit,
		Offset:  f.Offset,
		HasNext: f.Offset+len(items) < total,
	})
}
