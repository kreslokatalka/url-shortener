package delete_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/delete"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/service/delete/mocks"
)

func TestDeleteHandler(t *testing.T) {
	cases := []struct {
		name         string
		alias        string
		path         string
		wantStatus   int
		countDeleted int64
		respError    string
		mockError    error
	}{
		{
			name:         "Success",
			alias:        "test_alias",
			path:         "/test_alias",
			wantStatus:   http.StatusOK,
			countDeleted: 1,
		},
		{
			name:       "Empty alias",
			alias:      "",
			path:       "/",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "DeleteURL Error",
			alias:      "test_alias",
			path:       "/test_alias",
			wantStatus: http.StatusBadRequest,
			respError:  "failed to get url",
			mockError:  errors.New("unexpected error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlDeleterMock := mocks.NewURLDeleter(t)

			if tc.respError != "" || tc.countDeleted != 0 {
				urlDeleterMock.On("DeleteURL", tc.alias).
					Return(tc.countDeleted, tc.mockError).
					Once()
			}

			r := chi.NewRouter()
			r.Delete("/{alias}", delete.New(slogdiscard.NewDiscardLogger(), urlDeleterMock))

			req, err := http.NewRequest(http.MethodDelete, tc.path, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)

			if tc.wantStatus != http.StatusOK {
				return
			}

			var resp delete.Response

			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			require.Equal(t, tc.respError, resp.Error)

			if tc.respError == "" {
				require.Equal(t, tc.countDeleted, resp.CountDeleted)
			}

			urlDeleterMock.AssertExpectations(t)
		})
	}
}
