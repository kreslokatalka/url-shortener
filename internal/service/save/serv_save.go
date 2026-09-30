package save

import (
	"context"
	"errors"
	"log/slog"
	"url-shortener/internal/lib/random"
)

var ErrParseAliasLength = errors.New("failed to parse ALIAS_LENGTH")

//go:generate go run github.com/vektra/mockery/v2@latest --dir . --name URLSaver --output ./mocks
type URLSaver interface {
	SaveURL(ctx context.Context, urlToSave string, alias string) (int64, error)
}

type Service struct {
	urlSaver    URLSaver
	log         *slog.Logger
	aliasLength int
}

func NewService(urlSaver URLSaver, logger *slog.Logger, aliasLength int) *Service {
	return &Service{urlSaver: urlSaver, log: logger, aliasLength: aliasLength}
}

func (s *Service) Save(ctx context.Context, urlToSave, alias, ReqID string) (string, error) {
	const fn = "handlers.url.save.New"

	s.log = s.log.With(
		slog.String("fn", fn),
		slog.String("request_id", ReqID),
	)

	if alias == "" {
		if s.aliasLength <= 0 {
			return "", ErrParseAliasLength
		}
		alias = random.NewRandomString(s.aliasLength)
	}

	if _, err := s.urlSaver.SaveURL(ctx, urlToSave, alias); err != nil {
		return "", err
	}

	return alias, nil
}
