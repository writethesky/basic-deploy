package http

import (
	"io/ioutil"
	"net/http"
)

func Get(url string, header map[string]string) (statusCode int, body []byte, err error) {
	return do(http.MethodGet, url, header)
}

func Delete(url string, header map[string]string) (statusCode int, body []byte, err error) {
	return do(http.MethodDelete, url, header)
}

func do(method, url string, header map[string]string) (statusCode int, body []byte, err error) {
	req, err := http.NewRequest(method, url, nil)
	if nil != err {
		return
	}
	for key, value := range header {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if nil != err {
		return
	}

	body, err = ioutil.ReadAll(res.Body)
	if nil != err {
		return
	}
	defer res.Body.Close()

	statusCode = res.StatusCode

	return
}
