package update_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/update"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/service/update/mocks"
)

type contextKey struct{}

type updateTestCase struct {
	name             string
	input            string
	mockError        error
	wantUpdateCalled bool
	wantNewURL       string
	wantNewAlias     string
	wantAlias        string
	countUpdated     int64
	wantStatus       int
	wantError        string
}

func runUpdateTest(t *testing.T, tc updateTestCase) {
	t.Helper()

	urlUpdaterMock := mocks.NewURLUpdater(t)
	ctx := context.WithValue(context.Background(), contextKey{}, "request")

	if tc.wantUpdateCalled {
		urlUpdaterMock.On("UpdateURL", mock.MatchedBy(func(got context.Context) bool {
			return got.Value(contextKey{}) == "request"
		}), tc.wantNewURL, tc.wantNewAlias, tc.wantAlias).
			Return(tc.countUpdated, tc.mockError).
			Once()
	} else {
		urlUpdaterMock.AssertNotCalled(t, "UpdateURL", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	}

	router := chi.NewRouter()
	router.Patch("/url/{alias}", update.New(slogdiscard.NewDiscardLogger(), urlUpdaterMock))

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, "/url/test_alias", bytes.NewReader([]byte(tc.input)))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	require.Equal(t, tc.wantStatus, rr.Code)

	var resp update.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	require.Equal(t, tc.wantError, resp.Error)

	if tc.wantStatus == http.StatusOK {
		require.Equal(t, tc.countUpdated, resp.CountUpdated)
	}

	urlUpdaterMock.AssertExpectations(t)
}

func TestUpdate(t *testing.T) {
	cases := []updateTestCase{
		{
			name:             "Success with new alias",
			input:            `{"alias": "test_alias", "newurl": "https://google.com", "newalias": "new_alias"}`,
			wantUpdateCalled: true,
			wantNewURL:       "https://google.com",
			wantNewAlias:     "new_alias",
			wantAlias:        "test_alias",
			countUpdated:     1,
			wantStatus:       http.StatusOK,
		},
		{
			name:             "Success with empty new alias",
			input:            `{"alias": "test_alias", "newurl": "https://google.com", "newalias": ""}`,
			wantUpdateCalled: true,
			wantNewURL:       "https://google.com",
			wantNewAlias:     "test_alias",
			wantAlias:        "test_alias",
			countUpdated:     1,
			wantStatus:       http.StatusOK,
		},
		{
			name:       "Truncated JSON",
			input:      `{"alias": "test_alias", "newurl": "https://google.com"`,
			wantStatus: http.StatusBadRequest,
			wantError:  "failed to decode request body",
		},
		{
			name:       "Not a JSON",
			input:      `not a json at all`,
			wantStatus: http.StatusBadRequest,
			wantError:  "failed to decode request body",
		},
		{
			name:       "Wrong field type",
			input:      `{"alias": 123, "newurl": []}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "failed to decode request body",
		},
		{
			name:       "Empty body",
			input:      ``,
			wantStatus: http.StatusBadRequest,
			wantError:  "failed to decode request body",
		},
		{
			name:       "Empty alias",
			input:      `{"alias": "", "newurl": "https://google.com"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "field Alias is a required field",
		},
		{
			name:       "Empty new URL",
			input:      `{"alias": "test_alias", "newurl": ""}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "field NewURL is a required field",
		},
		{
			name:       "Invalid new URL",
			input:      `{"alias": "test_alias", "newurl": "not a url"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "field NewURL is not a valid URL",
		},
		{
			name:             "UpdateURL Error",
			input:            `{"alias": "test_alias", "newurl": "https://google.com", "newalias": "new_alias"}`,
			wantUpdateCalled: true,
			wantNewURL:       "https://google.com",
			wantNewAlias:     "new_alias",
			wantAlias:        "test_alias",
			mockError:        errors.New("unexpected error"),
			wantStatus:       http.StatusInternalServerError,
			wantError:        "failed to update url",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runUpdateTest(t, tc)
		})
	}
}
