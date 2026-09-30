package save_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/service/save/mocks"
	"url-shortener/internal/storage"
)

type contextKey struct{}

type saveTestCase struct {
	name           string
	input          string
	aliasLength    int
	mockError      error
	wantSaveCalled bool
	wantURL        string
	wantStatus     int
	wantError      string
	wantAlias      string
	wantAnyAlias   bool
}

func runSaveTest(t *testing.T, tc saveTestCase) {
	t.Helper()

	urlSaverMock := mocks.NewURLSaver(t)
	ctx := context.WithValue(context.Background(), contextKey{}, "request")

	if tc.wantSaveCalled {
		urlSaverMock.On("SaveURL", mock.MatchedBy(func(got context.Context) bool {
			return got.Value(contextKey{}) == "request"
		}), tc.wantURL, mock.AnythingOfType("string")).
			Return(int64(1), tc.mockError).
			Once()
	} else {
		urlSaverMock.AssertNotCalled(t, "SaveURL", mock.Anything, mock.Anything, mock.Anything)
	}

	handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock, tc.aliasLength)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "/save", bytes.NewReader([]byte(tc.input)))
	require.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	require.Equal(t, tc.wantStatus, rr.Code)

	var resp save.Response
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

	require.Equal(t, tc.wantError, resp.Error)

	if tc.wantStatus == http.StatusCreated {
		if tc.wantAnyAlias {
			require.NotEmpty(t, resp.Alias)
		} else {
			require.Equal(t, tc.wantAlias, resp.Alias)
		}
	}

	urlSaverMock.AssertExpectations(t)
}

func TestSave(t *testing.T) {
	cases := []saveTestCase{
		{
			name:           "Success",
			input:          `{"url": "https://google.com", "alias": "test_alias"}`,
			aliasLength:    5,
			wantSaveCalled: true,
			wantURL:        "https://google.com",
			wantStatus:     http.StatusCreated,
			wantAlias:      "test_alias",
		},
		{
			name:           "Empty alias with generated alias",
			input:          `{"url": "https://google.com", "alias": ""}`,
			aliasLength:    5,
			wantSaveCalled: true,
			wantURL:        "https://google.com",
			wantStatus:     http.StatusCreated,
			wantAnyAlias:   true,
		},
		{
			name:        "Truncated JSON",
			input:       `{"url": "https://google.com", "alias": "test_alias"`,
			aliasLength: 5,
			wantStatus:  http.StatusBadRequest,
			wantError:   "failed to decode request body",
		},
		{
			name:        "Not a JSON",
			input:       `not a json at all`,
			aliasLength: 5,
			wantStatus:  http.StatusBadRequest,
			wantError:   "failed to decode request body",
		},
		{
			name:        "Wrong field type",
			input:       `{"url": 123, "alias": []}`,
			aliasLength: 5,
			wantStatus:  http.StatusBadRequest,
			wantError:   "failed to decode request body",
		},
		{
			name:        "Empty body",
			input:       ``,
			aliasLength: 5,
			wantStatus:  http.StatusBadRequest,
			wantError:   "failed to decode request body",
		},
		{
			name:        "Empty URL",
			input:       `{"url": "", "alias": "some_alias"}`,
			aliasLength: 5,
			wantStatus:  http.StatusBadRequest,
			wantError:   "field URL is a required field",
		},
		{
			name:        "Invalid URL",
			input:       `{"url": "some invalid URL", "alias": "some_alias"}`,
			aliasLength: 5,
			wantStatus:  http.StatusBadRequest,
			wantError:   "field URL is not a valid URL",
		},
		{
			name:           "Alias already exists",
			input:          `{"url": "https://google.com", "alias": "test_alias"}`,
			aliasLength:    5,
			wantSaveCalled: true,
			wantURL:        "https://google.com",
			mockError:      storage.ErrAliasExists,
			wantStatus:     http.StatusConflict,
			wantError:      "alias already exists",
		},
		{
			name:           "SaveURL Error",
			input:          `{"url": "https://google.com", "alias": "test_alias"}`,
			aliasLength:    5,
			wantSaveCalled: true,
			wantURL:        "https://google.com",
			mockError:      errors.New("unexpected error"),
			wantStatus:     http.StatusInternalServerError,
			wantError:      "failed to add url",
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runSaveTest(t, tc)
		})
	}
}
