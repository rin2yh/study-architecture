package handler

import (
	"context"

	"github.com/rin2yh/study-architecture/server/member/api"
	"github.com/rin2yh/study-architecture/server/member/internal/repository"
)

type Handler struct {
	repo repository.MemberRepository
}

var _ api.StrictServerInterface = (*Handler)(nil)

func New(repo repository.MemberRepository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetHealthz(_ context.Context, _ api.GetHealthzRequestObject) (api.GetHealthzResponseObject, error) {
	return api.GetHealthz200JSONResponse{Status: "ok"}, nil
}

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
