package save_test

import (
	"bytes"
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

func TestSave_Success(t *testing.T) {
	cases := []struct {
		name        string
		alias       string
		url         string
		aliasLength int
	}{
		{
			name:        "Success",
			alias:       "test_alias",
			url:         "https://google.com",
			aliasLength: 5,
		},
		{
			name:        "Empty alias with generated alias",
			alias:       "",
			url:         "https://google.com",
			aliasLength: 5,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlSaverMock := mocks.NewURLSaver(t)

			urlSaverMock.On("SaveURL", tc.url, mock.AnythingOfType("string")).
				Return(int64(1), nil).
				Once()

			handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock, tc.aliasLength)

			input := `{"url": "` + tc.url + `", "alias": "` + tc.alias + `"}`

			req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, http.StatusCreated, rr.Code)

			var resp save.Response
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			require.Equal(t, "", resp.Error)
			require.NotEmpty(t, resp.Alias)
			if tc.alias != "" {
				require.Equal(t, tc.alias, resp.Alias)
			}

			urlSaverMock.AssertExpectations(t)
		})
	}
}

func TestSave_BadRequest(t *testing.T) {
	t.Run("Broken JSON", func(t *testing.T) {
		t.Parallel()

		brokenJSON := []struct {
			name  string
			input string
		}{
			{name: "Truncated JSON", input: `{"url": "https://google.com", "alias": "test_alias"`},
			{name: "Not a JSON", input: `not a json at all`},
			{name: "Wrong field type", input: `{"url": 123, "alias": []}`},
			{name: "Empty body", input: ``},
		}

		for _, tc := range brokenJSON {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				urlSaverMock := mocks.NewURLSaver(t)
				urlSaverMock.AssertNotCalled(t, "SaveURL", mock.Anything, mock.Anything)

				handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock, 5)

				req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(tc.input)))
				require.NoError(t, err)

				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				require.Equal(t, http.StatusBadRequest, rr.Code)

				var resp save.Response
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

				require.Equal(t, "failed to decode request body", resp.Error)
			})
		}
	})

	t.Run("Validation error", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name        string
			alias       string
			url         string
			aliasLength int
			respError   string
		}{
			{
				name:        "Empty URL",
				url:         "",
				alias:       "some_alias",
				aliasLength: 5,
				respError:   "field URL is a required field",
			},
			{
				name:        "Invalid URL",
				url:         "some invalid URL",
				alias:       "some_alias",
				aliasLength: 5,
				respError:   "field URL is not a valid URL",
			},
		}

		for _, tc := range cases {
			tc := tc

			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				urlSaverMock := mocks.NewURLSaver(t)
				urlSaverMock.AssertNotCalled(t, "SaveURL", mock.Anything, mock.Anything)

				handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock, tc.aliasLength)

				input := `{"url": "` + tc.url + `", "alias": "` + tc.alias + `"}`

				req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
				require.NoError(t, err)

				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				require.Equal(t, http.StatusBadRequest, rr.Code)

				var resp save.Response
				require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

				require.Equal(t, tc.respError, resp.Error)
			})
		}
	})
}

func TestSave_Conflict(t *testing.T) {
	t.Run("Alias already exists", func(t *testing.T) {
		t.Parallel()

		urlSaverMock := mocks.NewURLSaver(t)
		urlSaverMock.On("SaveURL", "https://google.com", "test_alias").
			Return(int64(0), storage.ErrAliasExists).
			Once()

		handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock, 5)

		input := `{"url": "https://google.com", "alias": "test_alias"}`

		req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		require.Equal(t, http.StatusConflict, rr.Code)

		var resp save.Response
		require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

		require.Equal(t, "alias already exists", resp.Error)

		urlSaverMock.AssertExpectations(t)
	})
}

func TestSave_Other(t *testing.T) {
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
			name:        "Empty alias with zero aliasLength",
			alias:       "",
			url:         "https://google.com",
			aliasLength: 0,
			wantStatus:  http.StatusInternalServerError,
			respError:   "failed to parse ALIAS_LENGTH",
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

			if tc.respError == "failed to parse ALIAS_LENGTH" {
				urlSaverMock.AssertNotCalled(t, "SaveURL", mock.Anything, mock.Anything)
			} else {
				urlSaverMock.On("SaveURL", tc.url, mock.AnythingOfType("string")).
					Return(int64(1), tc.mockError).
					Once()
			}

			handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock, tc.aliasLength)

			input := `{"url": "` + tc.url + `", "alias": "` + tc.alias + `"}`

			req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)

			var resp save.Response
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))

			require.Equal(t, tc.respError, resp.Error)

			urlSaverMock.AssertExpectations(t)
		})
	}
}
