package save

import (
	"errors"
	"log/slog"
	"net/http"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/service/save"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator"
)

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

func New(log *slog.Logger, urlSaver save.URLSaver, aliasLength int) http.HandlerFunc {
	svc := save.NewService(urlSaver, log, aliasLength)

	return func(w http.ResponseWriter, r *http.Request) {
		const fn = "handlers.url.save.New"

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
		alias, err := svc.Save(r.Context(), req.URL, req.Alias, middleware.GetReqID(r.Context()))

		if errors.Is(err, save.ErrParseAliasLength) {
			log.Error("failed to parse ALIAS_LENGTH", sl.Err(err))
			render.Status(r, 500)
			render.JSON(w, r, resp.Error("failed to parse ALIAS_LENGTH"))
			return
		}

		if errors.Is(err, storage.ErrAliasExists) {
			log.Info("alias already exists", slog.String("alias", req.Alias))
			render.Status(r, 409)
			render.JSON(w, r, resp.Error("alias already exists"))
			return
		}

		if err != nil {
			log.Error("failed to add url", sl.Err(err))
			render.Status(r, 500)
			render.JSON(w, r, resp.Error("failed to add url"))
			return
		}

		log.Info("url added", slog.String("alias", alias))

		responseOK(w, r, alias)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, alias string) {
	render.Status(r, 201)
	render.JSON(w, r, Response{
		Response: resp.OK(),
		Alias:    alias,
	})
}
