/*
Copyright 2022 FerryProxy Authors.

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
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type Client struct {
	baseURL string
	client  http.Client
}

func NewClient(baseUrl string) *Client {
	return &Client{baseURL: baseUrl}
}

func (c *Client) Get(ctx context.Context) (int32, error) {
	resp, err := c.client.Get(c.baseURL + "/unused")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("response %s:\n%s", http.StatusText(resp.StatusCode), string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	port, err := strconv.ParseInt(string(body), 10, 64)
	if err != nil {
		return 0, err
	}
	return int32(port), nil
}
