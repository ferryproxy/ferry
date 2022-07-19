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

package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"

	"golang.org/x/crypto/ssh"
)

func GetKey() (identity, authorized string, err error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	signer, err := ssh.NewSignerFromSigner(key)
	if err != nil {
		return "", "", err
	}

	k, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", "", err
	}
	p := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: k,
	})

	identity = base64.URLEncoding.EncodeToString(p)
	authorized = base64.URLEncoding.EncodeToString(ssh.MarshalAuthorizedKey(signer.PublicKey()))

	return identity, authorized, nil
}
