package redirect

import (
	"errors"
	st "url-shortener/internal/storage"
)

var (
	ErrAliasEmpty = errors.New("Alias empty")
)

//go:generate go run github.com/vektra/mockery/v2@latest --dir . --name URLGetter --output ./mocks
type URLGetter interface {
	GetURL(alias string) (string, error)
}

type Service struct {
	urlGetter URLGetter
}

func NewService(urlGetter URLGetter) *Service {
	return &Service{urlGetter: urlGetter}
}

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
