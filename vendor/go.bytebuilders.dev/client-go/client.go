/*
Copyright 2019 AppsCode Inc.
Copyright 2014 The Gogs Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

var jsonHeader = http.Header{"Content-Type": []string{"application/json;charset=UTF-8"}}

// Version return the library version
func Version() string {
	return "v0.0.1"
}

// Client represents a ByteBuilders api client
type Client struct {
	url         string
	accessToken string
	license     string
	sudo        string
	client      *http.Client
}

// NewClient initializes and returns a API client.
func NewClient(accessToken, license string, baseURL ...string) *Client {
	url := "https://byte.builders"
	if len(baseURL) > 0 {
		url = baseURL[0]
	}
	return &Client{
		url:         strings.TrimSuffix(url, "/"),
		accessToken: accessToken,
		license:     license,
		client:      &http.Client{},
	}
}

// NewClientWithHTTP creates an API client with a custom http client
func NewClientWithHTTP(httpClient *http.Client, accessToken, license string, baseURL ...string) *Client {
	client := NewClient(accessToken, license, baseURL...)
	client.client = httpClient
	return client
}

// SetHTTPClient replaces default http.Client with user given one.
func (c *Client) SetHTTPClient(client *http.Client) {
	c.client = client
}

// SetSudo sets username to impersonate.
func (c *Client) SetSudo(sudo string) {
	c.sudo = sudo
}

func (c *Client) doRequest(method, path string, header http.Header, body io.Reader) (*http.Response, error) {
	path = strings.TrimPrefix(path, "/")
	req, err := http.NewRequest(method, c.url+"/api/v1/"+path, body)
	if err != nil {
		return nil, err
	}
	if len(c.accessToken) != 0 {
		req.Header.Set("Authorization", "token "+c.accessToken)
	}
	if len(c.license) != 0 {
		req.Header.Add("Authorization", "JWT "+c.license)
	}
	if c.sudo != "" {
		req.Header.Set("Sudo", c.sudo)
	}
	for k, v := range header {
		req.Header[k] = v
	}

	return c.client.Do(req)
}

func (c *Client) getResponse(method, path string, header http.Header, body io.Reader) ([]byte, error) {
	resp, err := c.doRequest(method, path, header, body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return nil, errors.New("401 Unauthorized")
	case http.StatusForbidden:
		return nil, errors.New("403 Forbidden")
	case http.StatusNotFound:
		return nil, errors.New("404 Not Found")
	case http.StatusConflict:
		return nil, errors.New("409 Conflict")
	case http.StatusUnprocessableEntity:
		return nil, fmt.Errorf("422 Unprocessable Entity: %s", string(data))
	}

	if resp.StatusCode/100 != 2 {
		errMap := make(map[string]interface{})
		if err = json.Unmarshal(data, &errMap); err != nil {
			// when the JSON can't be parsed, data was probably empty or a plain string,
			// so we try to return a helpful error anyway
			return nil, fmt.Errorf("Unknown API Error: %d %s", resp.StatusCode, string(data))
		}
		return nil, errors.New(errMap["message"].(string))
	}

	return data, nil
}

func (c *Client) getParsedResponse(method, path string, header http.Header, body io.Reader, obj interface{}) error {
	data, err := c.getResponse(method, path, header, body)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, obj)
}

func (c *Client) getStatusCode(method, path string, header http.Header, body io.Reader) (int, error) {
	resp, err := c.doRequest(method, path, header, body)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}
