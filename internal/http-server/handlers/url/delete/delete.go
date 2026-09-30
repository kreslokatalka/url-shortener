package delete

import (
	"errors"
	"log/slog"
	"net/http"

	resp "url-shortener/internal/lib/api/response"
	del "url-shortener/internal/service/delete"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/go-chi/render"
)

type Response struct {
	resp.Response
	CountDeleted int64 `json:"countDeleted"`
}

// New принимает HTTP-запрос, валидирует его и делегирует обработку в Service.
func New(log *slog.Logger, urlDeleter del.URLDeleter) http.HandlerFunc {
	svc := del.NewService(urlDeleter)

	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "handlers.url.delete.New"

		log = log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		countDeleted, err := svc.Delete(r.Context(), alias)

		if errors.Is(err, del.ErrAliasEmpty) {
			log.Info("alias empty")
			render.Status(r, 404)
			render.JSON(w, r, resp.Error("alias empty"))
			return
		}

		if err != nil {
			log.Error("failed to get url", "alias", alias)
			render.Status(r, 500)
			render.JSON(w, r, resp.Error("failed to get url"))
			return
		}

		log.Info("deleted url", slog.String("alias", alias), slog.Int64("count_deleted", countDeleted))

		responseOK(w, r, countDeleted)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, countDeleted int64) {
	render.JSON(w, r, Response{
		Response:     resp.OK(),
		CountDeleted: countDeleted,
	})
}
