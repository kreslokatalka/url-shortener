package redirect_test

import (
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

	if tc.wantGetCalled {
		urlGetterMock.On("GetURL", tc.alias).
			Return(tc.url, tc.mockError).
			Once()
	} else {
		urlGetterMock.AssertNotCalled(t, "GetURL", mock.Anything)
	}

	r := chi.NewRouter()
	r.Get("/{alias}", redirect.New(slogdiscard.NewDiscardLogger(), urlGetterMock))

	ts := httptest.NewServer(r)
	defer ts.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(ts.URL + "/" + tc.alias)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, tc.wantStatus, resp.StatusCode)

	if tc.wantStatus == http.StatusFound {
		require.Equal(t, tc.url, resp.Header.Get("Location"))
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
