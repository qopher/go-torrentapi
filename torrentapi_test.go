package torrentapi

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
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

var nrOfErrors int

func TestCall(t *testing.T) {
	testData := []struct {
		desc       string
		api        API
		getRes     func(string) (*APIResponse, error)
		renewToken func() (Token, error)
		want       TorrentResults
		wantErr    bool
	}{
		{
			desc: "Empty torrent response",
			api:  API{APIToken: Token{Token: "test", Expires: time.Now().Add(time.Second * 100)}},
			getRes: func(string) (*APIResponse, error) {
				return &APIResponse{Torrents: []byte("[]")}, nil
			},
			renewToken: func() (Token, error) {
				return Token{}, nil
			},
			want:    TorrentResults{},
			wantErr: false,
		},
		{
			desc: "First query returns error",
			api:  API{APIToken: Token{Token: "test", Expires: time.Now().Add(time.Second * 100)}},
			getRes: func(string) (*APIResponse, error) {
				return nil, errors.New("error")
			},
			renewToken: func() (Token, error) {
				return Token{}, nil
			},
			want:    TorrentResults{},
			wantErr: true,
		},
		{
			desc: "Got expired token response, second query OK",
			api:  API{APIToken: Token{Token: "test", Expires: time.Now().Add(time.Second * 100)}},
			getRes: func(string) (*APIResponse, error) {
				if nrOfErrors == 0 {
					nrOfErrors++
					return &APIResponse{Torrents: nil, Error: "error", ErrorCode: 4}, nil
				}
				return &APIResponse{Torrents: []byte("[]")}, nil
			},
			renewToken: func() (Token, error) {
				return Token{}, nil
			},
			want:    TorrentResults{},
			wantErr: false,
		},
		{
			desc: "Got expired token response, second query error",
			api:  API{APIToken: Token{Token: "test", Expires: time.Now().Add(time.Second * 100)}},
			getRes: func(string) (*APIResponse, error) {
				if nrOfErrors == 0 {
					nrOfErrors++
					return &APIResponse{Torrents: nil, Error: "error", ErrorCode: 4}, nil
				}
				return &APIResponse{Torrents: []byte("[]")}, errors.New("test")
			},
			renewToken: func() (Token, error) {
				return Token{}, nil
			},
			want:    TorrentResults{},
			wantErr: true,
		},
		{
			desc: "Token invalid, renew returns error",
			api:  API{APIToken: Token{}},
			getRes: func(string) (*APIResponse, error) {
				return nil, nil
			},
			renewToken: func() (Token, error) {
				return Token{}, errors.New("token error")
			},
			want:    TorrentResults{},
			wantErr: true,
		},
		{
			desc: "Got error in first response",
			api:  API{APIToken: Token{Token: "test", Expires: time.Now().Add(time.Second * 100)}},
			getRes: func(string) (*APIResponse, error) {
				return &APIResponse{Torrents: nil, Error: "error", ErrorCode: 5}, nil
			},
			renewToken: func() (Token, error) {
				return Token{}, nil
			},
			want:    TorrentResults{},
			wantErr: true,
		},
		{
			desc: "First reponse renew token,  error in second response",
			api:  API{APIToken: Token{Token: "test", Expires: time.Now().Add(time.Second * 100)}},
			getRes: func(string) (*APIResponse, error) {
				if nrOfErrors == 0 {
					nrOfErrors++
					return &APIResponse{Torrents: nil, Error: "error", ErrorCode: 4}, nil
				}
				return &APIResponse{Torrents: nil, Error: "error", ErrorCode: 5}, nil
			},
			renewToken: func() (Token, error) {
				return Token{}, nil
			},
			want:    TorrentResults{},
			wantErr: true,
		},
		{
			desc: "First reponse renew token,  error in renew",
			api:  API{APIToken: Token{Token: "test", Expires: time.Now().Add(time.Second * 100)}},
			getRes: func(string) (*APIResponse, error) {
				return &APIResponse{Torrents: nil, Error: "error", ErrorCode: 4}, nil
			},
			renewToken: func() (Token, error) {
				return Token{}, errors.New("error")
			},
			want:    TorrentResults{},
			wantErr: true,
		},
		{
			desc: "Error in unmarshal",
			api:  API{APIToken: Token{Token: "test", Expires: time.Now().Add(time.Second * 100)}},
			getRes: func(string) (*APIResponse, error) {
				return &APIResponse{Torrents: []byte("")}, nil
			},
			renewToken: func() (Token, error) {
				return Token{}, nil
			},
			want:    TorrentResults{},
			wantErr: true,
		},
		{
			desc: "No torrents found",
			api:  API{APIToken: Token{Token: "test", Expires: time.Now().Add(time.Second * 100)}},
			getRes: func(string) (*APIResponse, error) {
				return &APIResponse{Torrents: nil, Error: "error", ErrorCode: 20}, nil
			},
			renewToken: func() (Token, error) {
				return Token{}, nil
			},
			want:    nil,
			wantErr: false,
		},
	}

	for i, tc := range testData {
		nrOfErrors = 0
		getRes = tc.getRes
		renewToken = tc.renewToken
		got, err := tc.api.call()

		if (err == nil) != !tc.wantErr {
			t.Errorf("Test (%d) %s: API.call() %+v => got unexpected error %v", i, tc.desc, tc.api, err)
		}
		if err != nil {
			continue
		}
		fmt.Println(got == nil)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Test(%d) %s: API.call() %+v => got: %+v, want: %+v", i, tc.desc, tc.api, got, tc.want)
		}

	}
}
