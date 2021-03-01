package api

import (
	"io"
	"net/http"
	"path"
	"strings"
)

type API struct {
	ep          string
	key, secret string
	client      *http.Client
}

func New(key, secret, endpointV2 string, client *http.Client) *API {
	if client == nil {
		client = http.DefaultClient
	}

	return &API{
		ep:     strings.TrimRight(endpointV2, "/"),
		key:    key,
		secret: secret,
		client: client,
	}
}

func (api *API) URL(apiPath ...string) string {
	return api.ep + "/" + path.Join(apiPath...) + "/" // much wow, required trailing slash
}

func (api *API) sign(r *http.Request) error {
	return Sign(api.key, api.secret, r)
}

func (api *API) Do(r *http.Request) (*http.Response, error) {
	if err := api.sign(r); err != nil {
		return nil, err
	}

	return api.client.Do(r)
}

func (api *API) makeDo(method, url string, body io.Reader) (*http.Response, error) {
	r, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	return api.Do(r)
}

func (api *API) Post(url string, body io.Reader) (*http.Response, error) {
	return api.makeDo("POST", url, body)
}

func (api *API) Get(url string, body io.Reader) (*http.Response, error) {
	return api.makeDo("GET", url, body)
}
