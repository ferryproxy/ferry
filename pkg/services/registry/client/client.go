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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ferryproxy/ferry/pkg/ferryctl/kubectl"
	"github.com/ferryproxy/ferry/pkg/services/registry/models"
)

type Client struct {
	baseURL string
	client  http.Client
}

func NewClient(baseUrl string) *Client {
	return &Client{baseURL: baseUrl}
}

func (c *Client) Create(ctx context.Context, hubName string) error {
	kctl := kubectl.NewKubectl()

	key, err := kctl.GetSecretAuthorized(ctx)
	if err != nil {
		return err
	}

	token, err := kctl.GetToken(ctx)
	if err != nil {
		return err
	}

	rawKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}

	data := &models.JoinHub{
		HubName:       hubName,
		AuthorizedKey: string(rawKey),
		Token:         token,
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	resp, err := c.client.Post(c.baseURL+"/"+hubName, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("response %s:\n%s", http.StatusText(resp.StatusCode), string(body))
	}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = kctl.ApplyWithReader(ctx, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) IsExist(ctx context.Context, hubName string) (bool, error) {
	resp, err := c.client.Head(c.baseURL + "/" + hubName)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode == http.StatusCreated {
		return true, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}
	return false, fmt.Errorf("response %s:\n%s", http.StatusText(resp.StatusCode), string(body))
}
