package github

import (
	httpUtil "basic-deploy/http"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const host = "https://api.github.com"

var baseHeader = map[string]string{
	"Accept": "application/vnd.github.v3+json",
}

type errorResponse struct {
	Message string `json:"message"`
}

var _token string

func SetToken(token string) {
	_token = token
}

func get(subURL string) (body []byte, err error) {
	return do(http.MethodGet, subURL)
}

func delete(subURL string) (body []byte, err error) {
	return do(http.MethodDelete, subURL)
}

func do(method, subURL string) (body []byte, err error) {
	var statusCode int
	if _token != "" {
		baseHeader["Authorization"] = fmt.Sprintf("token %s", _token)
	}
	switch method {
	case http.MethodGet:
		statusCode, body, err = httpUtil.Get(fmt.Sprintf("%s/%s", host, subURL), baseHeader)
	case http.MethodDelete:
		statusCode, body, err = httpUtil.Delete(fmt.Sprintf("%s/%s", host, subURL), baseHeader)
	default:
		return nil, errors.New("an unsupported HTTP method")
	}

	if nil != err {
		return
	}

	if statusCode < 200 || statusCode >= 400 {
		response := new(errorResponse)
		err = json.Unmarshal(body, response)
		if nil != err {
			return
		}
		err = errors.New(response.Message)
		return
	}

	return
}
