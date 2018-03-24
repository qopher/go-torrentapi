// Package torrentapi provides simple and easy Golang interface for RARBG Torrent API v2 (https://torrentapi.org)
package torrentapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// Version of supported torrentapi.
	Version = 2.0

	// DefaultAPIURL is a default value for APIURL option.
	DefaultAPIURL = "https://torrentapi.org/pubapi_v2.php?"

	// Defaulta.tokenExpiration is a default value for TokenExpiration option (TorrentAPI exprires after 15 min, but let's expire it after 890 seconds just to be safe.
	DefaultTokenExpiration = time.Second * 890

	// DefaultRequestDelay is a default delay between requests.
	DefaultRequestDelay = time.Second * 2

	// DefaultMaxRetries is a default value for MaxRetries option.
	DefaultMaxRetries = 10
)

// Error codes returned by TorrentAPI.
const (
	tokenExpiredCode = 4
	noResultsCode    = 20
	imdbNotFound     = 10
)

// Token keeps token and it's expiration date.
type Token struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"-"`
}

// EpisodeInfo keepsinformation from "episode_info" key from results. Some of the fields may be empty.
type EpisodeInfo struct {
	ImDB       string `json:"imdb"`
	TvDB       string `json:"tvdb"`
	TvRage     string `json:"tvrage"`
	TheMovieDb string `json:"themoviedb"`
	AirDate    string `json:"airdate"`
	SeasonNum  string `json:"seasonnum"`
	EpisodeNum string `json:"epnum"`
	Title      string `json:"title"`
}

// TorrentResult keeps information about single torrent returned from TorrentAPI. Some of the fields may be empty.
type TorrentResult struct {
	Title       string      `json:"title"`
	Filename    string      `json:"filename"`
	Category    string      `json:"category"`
	Download    string      `json:"download"`
	Seeders     int         `json:"seeders"`
	Leechers    int         `json:"leechers"`
	Size        uint64      `json:"size"`
	PubDate     string      `json:"pubdate"`
	Ranked      int         `json:"ranked"`
	InfoPage    string      `json:"info_page"`
	EpisodeInfo EpisodeInfo `json:"episode_info"`
}

// TorrentResults represents multiple results.
type TorrentResults []TorrentResult

// APIResponse from Torrent API.
type APIResponse struct {
	Torrents  json.RawMessage `json:"torrent_results"`
	Error     string          `json:"error"`
	ErrorCode int             `json:"error_code"`
}

// IsValid Check if token is still valid.
func (t *Token) IsValid() bool {
	if t.Token == "" {
		return false
	}
	if time.Now().After(t.Expires) {
		return false
	}
	return true
}

// API provides interface to access Torrent API.
type API struct {
	client          *http.Client
	Query           string
	APIToken        Token
	categories      []int
	appID           string
	reqDelay        time.Duration
	tokenExpiration time.Duration
	url             string
	maxRetries      int
}

// SearchString adds search string to search query.
func (a *API) SearchString(query string) *API {
	a.Query += fmt.Sprintf("&search_string=%s", url.QueryEscape(query))
	return a
}

// Category adds category to search query.
func (a *API) Category(category int) *API {
	a.categories = append(a.categories, category)
	return a
}

// SearchTVDB adds TheTVDB id to search query.
func (a *API) SearchTVDB(seriesid string) *API {
	a.Query += fmt.Sprintf("&search_tvdb=%s", seriesid)
	return a
}

// SearchIMDb adds IMDb id to search query.
func (a *API) SearchIMDb(movieid string) *API {
	a.Query += fmt.Sprintf("&search_imdb=%s", movieid)
	return a
}

// SearchTheMovieDb adds TheMovieDb id to search query.
func (a *API) SearchTheMovieDb(movieid string) *API {
	a.Query += fmt.Sprintf("&search_themoviedb=%s", movieid)
	return a
}

// Format requests different results format, possible values json, json_extended. Please note that whith json format not all fields are populated in TorrentResult.
func (a *API) Format(format string) *API {
	a.Query += fmt.Sprintf("&format=%s", format)
	return a
}

// Limit adds limit to number of results.
func (a *API) Limit(limit int) *API {
	a.Query += fmt.Sprintf("&limit=%d", limit)
	return a
}

// Sort results based on seeders, leechers or last(default).
func (a *API) Sort(sort string) *API {
	a.Query += fmt.Sprintf("&sort=%s", sort)
	return a
}

// Ranked sets if returned results should be ranked.
func (a *API) Ranked(ranked bool) *API {
	if ranked {
		a.Query += "&ranked=1"
	} else {
		a.Query += "&ranked=0"
	}
	return a
}

// MinSeeders specify minimum number of seeders.
func (a *API) MinSeeders(minSeed int) *API {
	a.Query += fmt.Sprintf("&min_seeders=%d", minSeed)
	return a
}

// MinLeechers specify minimum number of leechers.
func (a *API) MinLeechers(minLeech int) *API {
	a.Query += fmt.Sprintf("&min_leechers=%d", minLeech)
	return a
}

// List lists the newest torrrents, this has to be last function in chain.
func (a *API) List() (TorrentResults, error) {
	a.Query += "&mode=list"
	return a.call()
}

// Search performs search, this has to be last function in chain.
func (a *API) Search() (TorrentResults, error) {
	a.Query += "&mode=search"
	return a.call()
}

// getResults sends query to TorrentAPI and fetch the response.
func (a *API) getResults(query string) (*APIResponse, error) {
	resp, err := a.makeRequest(query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var apiResponse APIResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResponse)
	return &apiResponse, err
}

// call calls API and processes response.
func (a *API) call() (TorrentResults, error) {
	defer a.initQuery()
	if !a.APIToken.IsValid() {
		var err error
		a.APIToken, err = a.renewToken()
		if err != nil {
			return nil, err
		}
	}
	if len(a.categories) > 0 {
		categories := make([]string, len(a.categories))
		for i, c := range a.categories {
			categories[i] = strconv.Itoa(c)
		}
		a.Query += fmt.Sprintf("&category=%s", strings.Join(categories, ";"))
	}
	query := fmt.Sprintf("%s&token=%s%s&app_id=%s", a.url, a.APIToken.Token, a.Query, a.appID)
	apiResponse, err := a.getResults(query)
	if err != nil {
		return nil, err
	}
	data, err := a.processResponse(apiResponse)
	if err != nil {
		if _, ok := err.(*expiredTokenError); ok {
			// Token expired, renew it and try again
			a.APIToken, err = a.renewToken()
			if err != nil {
				return nil, err
			}
			apiResponse, err = a.getResults(query)
			if err != nil {
				return nil, err
			}
			return a.processResponse(apiResponse)
		}
	}
	return data, err
}

type expiredTokenError struct {
	s string
}

func (e expiredTokenError) Error() string {
	return e.s
}

// Process JSON data received from TorrentAPI
func (a *API) processResponse(apiResponse *APIResponse) (TorrentResults, error) {
	var data TorrentResults
	if apiResponse.Torrents != nil {
		// We have valid results
		if err := json.Unmarshal(apiResponse.Torrents, &data); err != nil {
			return nil, fmt.Errorf("query: %s, Error: %s", a.Query, err.Error())
		}
		return data, nil
	} else if apiResponse.Error != "" {
		// There was API error
		switch ec := apiResponse.ErrorCode; ec {
		// Token expired
		case tokenExpiredCode:
			return nil, &expiredTokenError{s: "expired token"}
		// No IMDb id found
		case imdbNotFound:
			return nil, nil
		// No torrents found
		case noResultsCode:
			return nil, nil
		default:
			return nil, fmt.Errorf("query: %s, Error: %s, Error code: %d)", a.Query, apiResponse.Error, ec)
		}
	}
	// It shouldn't happen
	return nil, fmt.Errorf("query: %s, Unknown error, got response: %v", a.Query, apiResponse)
}

// initQuery cleans query state.
func (a *API) initQuery() {
	a.categories = a.categories[:0]
	a.Query = ""
}

// RenewToken fetches new token.
func (a *API) renewToken() (Token, error) {
	var token Token
	resp, err := a.makeRequest(a.url + fmt.Sprintf("get_token=get_token&app_id=%s", a.appID))
	if err != nil {
		return token, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return token, fmt.Errorf("error decoding token: %v", err)
	}
	token.Expires = time.Now().Add(a.tokenExpiration)
	return token, nil
}

// makeRequest performs request with the provided query.
func (a *API) makeRequest(query string) (*http.Response, error) {
	maxAttempts := a.maxRetries
	for {
		maxAttempts--
		req, err := http.NewRequest("GET", query, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create http request: %v", err)
		}
		req.Header.Set("User-Agent", "go-torrentAPI/1.0")
		resp, err := a.client.Do(req)
		if err != nil {
			return nil, err
		}
		switch st := resp.StatusCode; st {
		case http.StatusOK:
			return resp, nil
		case http.StatusTooManyRequests:
			if maxAttempts > 0 {
				time.Sleep(a.reqDelay)
				continue
			}
			return nil, errors.New("maximum number of attempts reached")
		default:
			return nil, fmt.Errorf("non 200-OK respose: Code(%d) Status(%s)", resp.StatusCode, resp.Status)
		}
	}

}

// Option is an interface used to set various options for API.
type Option interface {
	set(a *API)
}

type option func(a *API)

func (o option) set(a *API) {
	o(a)
}

// APIURL sets URL for TorrentAPI.
func APIURL(url string) Option {
	return option(func(a *API) {
		a.url = url
	})
}

// TokenExpiration sets time after token expires.
func TokenExpiration(d time.Duration) Option {
	return option(func(a *API) {
		a.tokenExpiration = d
	})
}

// RequestDelay sets delay between requests.
func RequestDelay(d time.Duration) Option {
	return option(func(a *API) {
		a.reqDelay = d
	})
}

// MaxRetries sets maximum retries after 429 Too Many Requests response.
func MaxRetries(r int) Option {
	return option(func(a *API) {
		a.maxRetries = r
	})
}

// Init Initializes API object, fetches new token and returns API instance.
func New(appID string, opts ...Option) (*API, error) {
	a := &API{
		client:          &http.Client{},
		appID:           appID,
		reqDelay:        DefaultRequestDelay,
		url:             DefaultAPIURL,
		maxRetries:      DefaultMaxRetries,
		tokenExpiration: DefaultTokenExpiration,
	}
	for _, o := range opts {
		o.set(a)
	}
	if !strings.HasSuffix(a.url, "?") {
		a.url += "?"
	}
	token, err := a.renewToken()
	if err != nil {
		return nil, fmt.Errorf("error renewing token: %v", err)
	}
	a.APIToken = token
	a.initQuery()
	return a, err
}
