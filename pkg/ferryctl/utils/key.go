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
