package torrentapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
)

func TestTokenIsValid(t *testing.T) {
	testData := []struct {
		desc  string
		token Token
		want  bool
	}{
		{
			desc:  "Valid Token",
			token: Token{Token: "test", Expires: time.Now().Add(time.Second * 100)},
			want:  true,
		},
		{
			desc:  "Expired Token",
			token: Token{Token: "test", Expires: time.Now().Add(time.Second * -100)},
			want:  false,
		},
		{
			desc:  "Empty Token",
			token: Token{Token: "", Expires: time.Now().Add(time.Second * 100)},
			want:  false,
		},
	}
	for i, tc := range testData {
		got := tc.token.IsValid()
		if got != tc.want {
			t.Errorf("Test(%d) %s: (%+v).IsValid() => got %v, want %v", i, tc.desc, tc.token, got, tc.want)
		}
	}
}

func performQueryStringTest(api *API, expectedResult string) (result bool, text string) {
	if !strings.HasSuffix(api.Query, expectedResult) {
		text = fmt.Sprintf("api.Query doesn't ends with requested string: %s, Query: %s", expectedResult, api.Query)
		return
	}
	return true, ""
}

func TestAPISearchString(t *testing.T) {
	api := new(API)
	api = api.SearchString("test")
	result, text := performQueryStringTest(api, "&search_string=test")
	if !result {
		t.Error(text)
	}
}

func TestAPICategory(t *testing.T) {
	api := new(API)
	api = api.Category(1)
	if len(api.categories) != 1 {
		t.Error("Categories not added")
	}
}

func TestAPISearchTVDB(t *testing.T) {
	api := new(API)
	api = api.SearchTVDB("123")
	result, text := performQueryStringTest(api, "&search_tvdb=123")
	if !result {
		t.Error(text)
	}
}

func TestAPISearchIMDb(t *testing.T) {
	api := new(API)
	api = api.SearchIMDb("tt123")
	result, text := performQueryStringTest(api, "&search_imdb=tt123")
	if !result {
		t.Error(text)
	}
}

func TestAPISearchTheMovieDb(t *testing.T) {
	api := new(API)
	api = api.SearchTheMovieDb("123")
	result, text := performQueryStringTest(api, "&search_themoviedb=123")
	if !result {
		t.Error(text)
	}
}

func TestAPIFormat(t *testing.T) {
	api := new(API)
	api = api.Format("super_format")
	result, text := performQueryStringTest(api, "&format=super_format")
	if !result {
		t.Error(text)
	}
}

func TestAPILimit(t *testing.T) {
	api := new(API)
	api = api.Limit(100)
	result, text := performQueryStringTest(api, "&limit=100")
	if !result {
		t.Error(text)
	}
}

func TestAPISort(t *testing.T) {
	api := new(API)
	api = api.Sort("seeders")
	result, text := performQueryStringTest(api, "&sort=seeders")
	if !result {
		t.Error(text)
	}
}

func TestAPIRankedTrue(t *testing.T) {
	api := new(API)
	api = api.Ranked(true)
	result, text := performQueryStringTest(api, "&ranked=1")
	if !result {
		t.Error(text)
	}
}

func TestAPIRankedFalse(t *testing.T) {
	api := new(API)
	api = api.Ranked(false)
	result, text := performQueryStringTest(api, "&ranked=0")
	if !result {
		t.Error(text)
	}
}

func TestAPIRankedMinSeeders(t *testing.T) {
	api := new(API)
	api = api.MinSeeders(100)
	result, text := performQueryStringTest(api, "&min_seeders=100")
	if !result {
		t.Error(text)
	}
}

func TestAPIRankedMinLeechers(t *testing.T) {
	api := new(API)
	api = api.MinLeechers(100)
	result, text := performQueryStringTest(api, "&min_leechers=100")
	if !result {
		t.Error(text)
	}
}

type fakeHandler struct {
	sync.Mutex
	cnt      int
	handlers []http.HandlerFunc
}

func (f *fakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.Lock()
	defer f.Unlock()
	fn := f.handlers[f.cnt%len(f.handlers)]
	f.cnt++
	fn(w, r)
}

func (f *fakeHandler) setHandlers(h []http.HandlerFunc) {
	f.Lock()
	defer f.Unlock()
	f.handlers = h
	f.cnt = 0
}

func fakeTokenHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"token": "some_token"}`)
}

func error500(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "some error", http.StatusInternalServerError)
}

func errorTooMany(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "too many requests error", http.StatusTooManyRequests)
}

func errorTokenExpired(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{
		"error": "token expired",
		"error_code": 4
	}`)
}

func errorNoResults(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{
		"error": "no results",
		"error_code": 20
	}`)
}

func okResponse(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"torrent_results": [{"title": "Movie"}]}`)
}

func garbage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "garbage")
}

func TestAPI(t *testing.T) {
	testData := []struct {
		desc     string
		handlers []http.HandlerFunc
		wantErr  bool
		want     TorrentResults
	}{
		{
			desc:     "error - garbage data",
			handlers: []http.HandlerFunc{garbage},
			wantErr:  true,
		},
		{
			desc:     "error 500",
			handlers: []http.HandlerFunc{error500},
			wantErr:  true,
		},
		{
			desc: "too many requests then 500",
			handlers: []http.HandlerFunc{
				errorTooMany,
				error500,
			},
			wantErr: true,
		},
		{
			desc: "token expired, then 500",
			handlers: []http.HandlerFunc{
				errorTokenExpired,
				error500,
			},
			wantErr: true,
		},
		{
			desc: "token expired, then no results",
			handlers: []http.HandlerFunc{
				errorTokenExpired,
				fakeTokenHandler,
				errorNoResults,
			},
		},
		{
			desc: "too many requests then OK",
			handlers: []http.HandlerFunc{
				errorTooMany,
				okResponse,
			},
			want: TorrentResults{{Title: "Movie"}},
		},
		{
			desc: "token expired then OK",
			handlers: []http.HandlerFunc{
				errorTokenExpired,
				fakeTokenHandler,
				okResponse,
			},
			want: TorrentResults{{Title: "Movie"}},
		},
		{
			desc: "token expired then too many then OK",
			handlers: []http.HandlerFunc{
				errorTokenExpired,
				errorTooMany,
				fakeTokenHandler,
				errorTooMany,
				okResponse,
			},
			want: TorrentResults{{Title: "Movie"}},
		},
		{
			desc: "OK",
			handlers: []http.HandlerFunc{
				okResponse,
			},
			want: TorrentResults{{Title: "Movie"}},
		},
	}
	fh := &fakeHandler{handlers: []http.HandlerFunc{fakeTokenHandler}}
	ts := httptest.NewServer(fh)
	defer ts.Close()
	for i, tc := range testData {
		t.Run(fmt.Sprintf("Test (%d) %s", i, tc.desc), func(t *testing.T) {
			// insert token first to the handlers as New always fetches the token.
			fh.setHandlers(append([]http.HandlerFunc{fakeTokenHandler}, tc.handlers...))
			a, err := New("test", APIURL(ts.URL), RequestDelay(0))
			if err != nil {
				t.Fatalf("Failed to start API: %v", err)
			}
			got, err := a.call()
			if (err != nil) != tc.wantErr {
				t.Errorf("unexpected error got (%v) want (%v)", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if diff := pretty.Compare(tc.want, got); diff != "" {
				t.Errorf("unexpected results: diff -want +got\n%s", diff)
			}
		})
	}
}
