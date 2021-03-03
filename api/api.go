package api

import (
	"errors"
	"fmt"
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

type Status struct {
	Status string      `json:"status"`
	Reason interface{} `json:"reason"`
}

var ErrUnknown = errors.New("unknown api error")

func (s Status) Error() error {
	if s.Status == "" {
		return nil
	}
	if s.Reason == nil {
		return ErrUnknown
	}

	switch v := s.Reason.(type) {
	case map[string]interface{}:
		_e, ok := v["__all__"]
		if !ok {
			return fmt.Errorf("%+v", v)
		}

		if e, ok := _e.(string); ok {
			return errors.New(e)
		}

		if e, ok := _e.([]interface{}); ok && len(e) != 0 {
			strs := make([]string, len(e))
			for i := range e {
				strs[i] = fmt.Sprintf("%v", e[i])
			}

			return errors.New(strings.Join(strs, ", "))
		}

		return fmt.Errorf("%+v", _e)
	}

	return fmt.Errorf("%+v", s.Reason)
}
