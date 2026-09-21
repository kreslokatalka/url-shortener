package save_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/service/save/mocks"
)

func TestSaveHandler(t *testing.T) {
	cases := []struct {
		name        string
		alias       string
		url         string
		aliasLength int
		wantStatus  int
		respError   string
		mockError   error
	}{
		{
			name:        "Success",
			alias:       "test_alias",
			url:         "https://google.com",
			aliasLength: 5,
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Empty alias",
			alias:       "",
			url:         "https://google.com",
			aliasLength: 0,
			wantStatus:  http.StatusInternalServerError,
			respError:   "failed to parse ALIAS_LENGTH",
		},
		{
			name:        "Empty alias with generated alias",
			alias:       "",
			url:         "https://google.com",
			aliasLength: 5,
			wantStatus:  http.StatusOK,
		},
		{
			name:        "Empty URL",
			url:         "",
			alias:       "some_alias",
			aliasLength: 5,
			wantStatus:  http.StatusBadRequest,
			respError:   "field URL is a required field",
		},
		{
			name:        "Invalid URL",
			url:         "some invalid URL",
			alias:       "some_alias",
			aliasLength: 5,
			wantStatus:  http.StatusBadRequest,
			respError:   "field URL is not a valid URL",
		},
		{
			name:        "SaveURL Error",
			alias:       "test_alias",
			url:         "https://google.com",
			aliasLength: 5,
			wantStatus:  http.StatusBadRequest,
			respError:   "failed to add url",
			mockError:   errors.New("unexpected error"),
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlSaverMock := mocks.NewURLSaver(t)

			// Мок настраивается только тогда, когда запрос реально доходит до сохранения:
			// при ошибке валидации и при неудачном разборе aliasLength хендлер выходит раньше.
			isValidationError := tc.respError == "field URL is a required field" ||
				tc.respError == "field URL is not a valid URL"
			isAliasLengthError := tc.respError == "failed to parse ALIAS_LENGTH"

			if !isValidationError && !isAliasLengthError {
				urlSaverMock.On("SaveURL", tc.url, mock.AnythingOfType("string")).
					Return(int64(1), tc.mockError).
					Once()
			}

			// aliasLength передаётся явно — тест не зависит от переменных окружения.
			handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock, tc.aliasLength)

			input := fmt.Sprintf(`{"url": "%s", "alias": "%s"}`, tc.url, tc.alias)

			req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)

			body := rr.Body.String()

			var resp save.Response

			require.NoError(t, json.Unmarshal([]byte(body), &resp))

			require.Equal(t, tc.respError, resp.Error)

		})
	}
}
