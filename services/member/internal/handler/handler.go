package handler

import (
	"context"

	"github.com/rin2yh/study-service-base-architecture/services/member/api"
	"github.com/rin2yh/study-service-base-architecture/services/member/internal/repository"
)

// Handler は oapi-codegen 生成の StrictServerInterface を実装する。
type Handler struct {
	repo repository.MemberRepository
}

var _ api.StrictServerInterface = (*Handler)(nil)

// New は handler を生成する。kessoku から repository.MemberRepository を注入される。
func New(repo repository.MemberRepository) *Handler {
	return &Handler{repo: repo}
}

// GetHealthz は liveness を返す（DB 非依存）。
func (h *Handler) GetHealthz(_ context.Context, _ api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: "ok"}, nil
}

// ListMembers は会員一覧を返す。DB の行を API 表現へ詰め替える。
func (h *Handler) ListMembers(ctx context.Context, _ api.ListMembersRequestObject) (api.ListMembersResponseObject, error) {
	rows, err := h.repo.ListMembers(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListMembers200JSONResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, api.Member{
			Id:          r.ID,
			Email:       r.Email,
			DisplayName: r.DisplayName,
			CreatedAt:   r.CreatedAt.Time,
		})
	}
	return out, nil
}
