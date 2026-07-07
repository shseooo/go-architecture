package rest

import (
	"context"
	"net/http"

	"github.com/shseooo/go-architecture/app/domain"
)

// MemberService is the member use-case contract the handler depends on.
type MemberService interface {
	Register(ctx context.Context, m *domain.Member) error
	GetByID(ctx context.Context, id int64) (domain.Member, error)
}

type MemberHandler struct {
	svc MemberService
}

func NewMemberHandler(svc MemberService) *MemberHandler {
	return &MemberHandler{svc: svc}
}

type memberRequest struct {
	Name    string         `json:"name"`
	Address domain.Address `json:"address"`
}

// Register godoc
// @Summary  회원 등록
// @Tags     members
// @Accept   json
// @Produce  json
// @Param    body  body      memberRequest  true  "회원 정보"
// @Success  201   {object}  rest.Envelope[domain.Member]
// @Failure  400   {object}  rest.ErrorEnvelope  "입력 검증 실패"
// @Failure  500   {object}  rest.ErrorEnvelope  "서버 에러"
// @Router   /members [post]
func (h *MemberHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req memberRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	m := &domain.Member{Name: req.Name, Address: req.Address}
	if err := h.svc.Register(r.Context(), m); err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, m)
}

// GetByID godoc
// @Summary  회원 조회
// @Tags     members
// @Produce  json
// @Param    id   path      int  true  "회원 ID"
// @Success  200  {object}  rest.Envelope[domain.Member]
// @Failure  400  {object}  rest.ErrorEnvelope  "잘못된 ID"
// @Failure  404  {object}  rest.ErrorEnvelope  "회원 없음"
// @Router   /members/{id} [get]
func (h *MemberHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	m, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, m)
}
