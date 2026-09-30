package redirect_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/redirect"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/service/redirect/mocks"
	"url-shortener/internal/storage"
)

type contextKey struct{}

type redirectTestCase struct {
	name          string
	alias         string
	url           string
	mockError     error
	wantGetCalled bool
	wantStatus    int
	wantError     string
}

func runRedirectTest(t *testing.T, tc redirectTestCase) {
	t.Helper()

	urlGetterMock := mocks.NewURLGetter(t)
	ctx := context.WithValue(context.Background(), contextKey{}, "request")

	if tc.wantGetCalled {
		urlGetterMock.On("GetURL", mock.MatchedBy(func(got context.Context) bool {
			return got.Value(contextKey{}) == "request"
		}), tc.alias).
			Return(tc.url, tc.mockError).
			Once()
	} else {
		urlGetterMock.AssertNotCalled(t, "GetURL", mock.Anything, mock.Anything)
	}

	r := chi.NewRouter()
	r.Get("/{alias}", redirect.New(slogdiscard.NewDiscardLogger(), urlGetterMock))

	req := httptest.NewRequest(http.MethodGet, "/"+tc.alias, nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, tc.wantStatus, rr.Code)

	if tc.wantStatus == http.StatusFound {
		require.Equal(t, tc.url, rr.Header().Get("Location"))
	}

	urlGetterMock.AssertExpectations(t)
}

func TestRedirect(t *testing.T) {
	cases := []redirectTestCase{
		{
			name:          "Success",
			alias:         "test_alias",
			url:           "https://www.google.com/",
			wantGetCalled: true,
			wantStatus:    http.StatusFound,
		},
		{
			name:       "Empty alias",
			alias:      "",
			wantStatus: http.StatusNotFound,
		},
		{
			name:          "URL not found",
			alias:         "test_alias",
			wantGetCalled: true,
			mockError:     storage.ErrURLNotFound,
			wantStatus:    http.StatusNotFound,
		},
		{
			name:          "GetURL Error",
			alias:         "test_alias",
			wantGetCalled: true,
			mockError:     errors.New("unexpected error"),
			wantStatus:    http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runRedirectTest(t, tc)
		})
	}
}
