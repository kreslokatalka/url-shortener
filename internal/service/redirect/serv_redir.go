package redirect

import (
	"errors"
	st "url-shortener/internal/storage"
)

var (
	ErrAliasEmpty = errors.New("Alias empty")
)

//go:generate go run github.com/vektra/mockery/v2@latest --dir . --name URLDeleter --output ./mocks
type URLGetter interface {
	GetURL(alias string) (string, error)
}

// Service содержит бизнес-логику удаления URL.
type Service struct {
	urlGetter URLGetter
}

// NewService создаёт новый Service для удаления URL.
func NewService(urlGetter URLGetter) *Service {
	return &Service{urlGetter: urlGetter}
}

// Delete удаляет URL по alias и возвращает количество удалённых записей.
func (g *Service) Get(alias string) (string, error) {

	if alias == "" {
		return "", ErrAliasEmpty
	}

	url, err := g.urlGetter.GetURL(alias)
	if errors.Is(err, st.ErrURLNotFound) {
		return "", st.ErrURLNotFound
	}
	return url, err
}
