package tests

import (
	"net/http"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gavv/httpexpect/v2"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/handlers/url/update"
	"url-shortener/internal/lib/api"
	"url-shortener/internal/lib/random"
)

const (
	host = "localhost:8082"
)

func getAuth(typeField string) string {

	if err := godotenv.Load("../.env"); err != nil {
		panic("Error loading .env file: " + err.Error())
	}

	authUser := os.Getenv("AUTH_USER")
	authPassword := os.Getenv("AUTH_PASSWORD")

	if authUser == "" || authPassword == "" {
		panic("AUTH_USER and AUTH_PASSWORD must be set in .env file")
	}

	if typeField == "user" {
		return authUser
	}
	return authPassword

}

func newExpect(t *testing.T) *httpexpect.Expect {
	t.Helper()

	u := url.URL{
		Scheme: "http",
		Host:   host,
	}

	return httpexpect.Default(t, u.String())
}

func TestURLShortener_HappyPath(t *testing.T) {
	e := newExpect(t)

	e.POST("/url").
		WithJSON(save.Request{
			URL:   gofakeit.URL(),
			Alias: random.NewRandomString(10),
		}).
		WithBasicAuth(getAuth("user"), getAuth("password")).
		Expect().
		Status(http.StatusCreated).
		JSON().Object().
		ContainsKey("alias")
}

func TestURLShortener_SaveUpdateRedirectRemove(t *testing.T) {
	testCases := []struct {
		name       string
		url        string
		alias      string
		wantStatus int
		error      string
	}{
		{
			name:       "Valid URL",
			url:        gofakeit.URL(),
			alias:      gofakeit.Word() + gofakeit.Word(),
			wantStatus: http.StatusCreated,
		},
		{
			name:       "Invalid URL",
			url:        "invalid_url",
			alias:      gofakeit.Word(),
			wantStatus: http.StatusBadRequest,
			error:      "field URL is not a valid URL",
		},
		{
			name:       "Empty Alias",
			url:        gofakeit.URL(),
			alias:      "",
			wantStatus: http.StatusCreated,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			e := newExpect(t)

			resp := e.POST("/url").
				WithJSON(save.Request{
					URL:   tc.url,
					Alias: tc.alias,
				}).
				WithBasicAuth(getAuth("user"), getAuth("password")).
				Expect().Status(tc.wantStatus).
				JSON().Object()

			if tc.error != "" {
				resp.NotContainsKey("alias")

				resp.Value("error").String().IsEqual(tc.error)

				return
			}

			alias := tc.alias

			if tc.alias != "" {
				resp.Value("alias").String().IsEqual(tc.alias)
			} else {
				resp.Value("alias").String().NotEmpty()

				alias = resp.Value("alias").String().Raw()
			}

			testRedirect(t, alias, tc.url)

			newURL := gofakeit.URL()
			newAlias := alias + "upd"

			testUpdate(t, e, alias, newURL, newAlias)

			testRedirect(t, newAlias, newURL)

			reqDel := e.DELETE("/"+path.Join("url", newAlias)).
				WithBasicAuth(getAuth("user"), getAuth("password")).
				Expect().Status(http.StatusOK).
				JSON().Object()

			reqDel.Value("countDeleted").Number().IsEqual(1)

		})
	}
}

func testUpdate(t *testing.T, e *httpexpect.Expect, alias, newURL, newAlias string) {
	t.Helper()

	resp := e.PATCH("/"+path.Join("url", alias)).
		WithJSON(update.Request{
			Alias:    alias,
			NewURL:   newURL,
			NewAlias: newAlias,
		}).
		WithBasicAuth(getAuth("user"), getAuth("password")).
		Expect().Status(http.StatusOK).
		JSON().Object()

	resp.Value("countUpdated").Number().IsEqual(1)
}

func testRedirect(t *testing.T, alias string, urlToRedirect string) {
	u := url.URL{
		Scheme: "http",
		Host:   host,
		Path:   alias,
	}

	redirectedToURL, err := api.GetRedirect(u.String())
	require.NoError(t, err)

	require.Equal(t, urlToRedirect, redirectedToURL)
}
