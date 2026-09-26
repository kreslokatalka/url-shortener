package update

import (
	"log/slog"
	"net/http"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/service/update"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
)

type Request struct {
	Alias    string `json:"alias" validate:"required"`
	NewURL   string `json:"newurl" validate:"required,url"`
	NewAlias string `json:"newalias"`
}

type Response struct {
	resp.Response
	CountUpdated int64 `json:"countUpdated"`
}

func New(log *slog.Logger, updaterURL update.URLUpdater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "handlers.url.update.New"

		log = log.With(
			slog.String("fn", fn),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request
		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.Status(r, 400)
			render.JSON(w, r, resp.Error("failed to decode request body"))
			return
		}

		log.Info("request body decoded", slog.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)
			log.Error("invalid request", sl.Err(err))
			render.Status(r, 400)
			render.JSON(w, r, resp.ValidationError(validateErr))
			return
		}

		alias := req.Alias
		newURL := req.NewURL
		newAlias := req.NewAlias
		countUpdated, err := updaterURL.UpdateURL(newURL, newAlias, alias)

		if err != nil {
			log.Error("failed to update url", "alias", alias, sl.Err(err))
			render.Status(r, 500)
			render.JSON(w, r, resp.Error("failed to update url"))
			return
		}

		log.Info("url updated", slog.String("alias", alias), slog.Int64("count_deleted", countUpdated))

		responseOK(w, r, countUpdated)

	}
}

func responseOK(w http.ResponseWriter, r *http.Request, countUpdated int64) {
	render.JSON(w, r, Response{
		Response:     resp.OK(),
		CountUpdated: countUpdated,
	})
}
