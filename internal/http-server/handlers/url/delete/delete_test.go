package delete_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/service/delete/mocks"
)

type contextKey struct{}

type deleteTestCase struct {
	name             string
	path             string
	alias            string
	mockError        error
	wantDeleteCalled bool
	countDeleted     int64
	wantStatus       int
	wantError        string
}

func runDeleteTest(t *testing.T, tc deleteTestCase) {
	t.Helper()

	urlDeleterMock := mocks.NewURLDeleter(t)
	ctx := context.WithValue(context.Background(), contextKey{}, "request")

	if tc.wantDeleteCalled {
		urlDeleterMock.On("DeleteURL", mock.MatchedBy(func(got context.Context) bool {
			return got.Value(contextKey{}) == "request"
		}), tc.alias).
			Return(tc.countDeleted, tc.mockError).
			Once()
	} else {
		urlDeleterMock.AssertNotCalled(t, "DeleteURL", mock.Anything, mock.Anything)
	}

	r := chi.NewRouter()
	r.Delete("/{alias}", delete.New(slogdiscard.NewDiscardLogger(), urlDeleterMock))

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, tc.path, nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	require.Equal(t, tc.wantStatus, rr.Code)

	if tc.wantStatus != http.StatusOK {
		urlDeleterMock.AssertExpectations(t)
		return
	}

	var resp delete.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	require.Equal(t, tc.wantError, resp.Error)
	require.Equal(t, tc.countDeleted, resp.CountDeleted)

	urlDeleterMock.AssertExpectations(t)
}

func TestDelete(t *testing.T) {
	cases := []deleteTestCase{
		{
			name:             "Success",
			path:             "/test_alias",
			alias:            "test_alias",
			wantDeleteCalled: true,
			countDeleted:     1,
			wantStatus:       http.StatusOK,
		},
		{
			name:       "Empty alias",
			path:       "/",
			wantStatus: http.StatusNotFound,
		},
		{
			name:             "DeleteURL Error",
			path:             "/test_alias",
			alias:            "test_alias",
			wantDeleteCalled: true,
			mockError:        errors.New("unexpected error"),
			wantStatus:       http.StatusInternalServerError,
			wantError:        "failed to get url",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runDeleteTest(t, tc)
		})
	}
}
