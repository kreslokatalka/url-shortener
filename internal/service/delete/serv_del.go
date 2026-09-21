package delete

import (
	"errors"
)

var (
	ErrAliasEmpty = errors.New("Alias empty")
)

//go:generate go run github.com/vektra/mockery/v2@latest --dir . --name URLDeleter --output ./mocks
type URLDeleter interface {
	DeleteURL(alias string) (int64, error)
}

// Service содержит бизнес-логику удаления URL.
type Service struct {
	urlDeleter URLDeleter
}

// NewService создаёт новый Service для удаления URL.
func NewService(urlDeleter URLDeleter) *Service {
	return &Service{urlDeleter: urlDeleter}
}

// Delete удаляет URL по alias и возвращает количество удалённых записей.
func (s *Service) Delete(alias string) (int64, error) {

	if alias == "" {
		return 0, ErrAliasEmpty
	}

	return s.urlDeleter.DeleteURL(alias)
}
