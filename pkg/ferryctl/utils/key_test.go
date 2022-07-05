package utils

import (
	"encoding/base64"
	"testing"
)

func Test_GenKey(t *testing.T) {
	identityKey, authorized, err := GetKey()
	if err != nil {
		t.Errorf("GetKey() error = %v", err)
		return
	}
	i, _ := base64.StdEncoding.DecodeString(identityKey)
	a, _ := base64.StdEncoding.DecodeString(authorized)
	t.Log(string(i))
	t.Log(string(a))
}
