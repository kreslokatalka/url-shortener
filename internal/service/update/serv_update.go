package update

import "log/slog"

//go:generate go run github.com/vektra/mockery/v2@latest --dir . --name URLSaver --output ./mocks
type URLUpdater interface {
	UpdateURL(newURL, NewAlias, alias string) (int64, error)
}

type Service struct {
	urlUpdater URLUpdater
	log        *slog.Logger
}

func NewService(urlupdater URLUpdater, logger *slog.Logger) *Service {
	return &Service{urlUpdater: urlupdater, log: logger}
}

func (s *Service) Update(NewURL, NewAlias, alias string) (int64, error) {
	if NewAlias == "" {
		NewAlias = alias
	}

	return s.urlUpdater.UpdateURL(NewURL, NewAlias, alias)
}
