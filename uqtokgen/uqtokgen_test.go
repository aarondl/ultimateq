package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go"
)

const testConfig = `
secret_key = "key"
nick = "hi"
username = "there"
realname = "friends"
[networks.test]
servers = ["test:6667"]
`

const testBadConfig = `
nick = "hi"
username = "there"
realname = "friends"
[networks.test]
servers = ["test:6667"]
`

func TestRun(t *testing.T) {
	t.Parallel()

	tok, err := run("name", strings.NewReader(testConfig))
	if err != nil {
		t.Fatal(err)
	}

	token, err := jwt.Parse(tok, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte("key"), nil
	})

	if err != nil {
		t.Error(err)
	}
	if !token.Valid {
		t.Error("Token should be valid")
	}
}

func TestGenerateToken(t *testing.T) {
	t.Parallel()

	var tok string
	var err error

	tok, err = generateToken("", "")
	if len(tok) > 0 {
		t.Error("tok should be empty")
	}
	if !hasErr(err, "provide an extension name") {
		t.Error("want extension name err, got:", err)
	}

	tok, err = generateToken("name", "")
	if len(tok) > 0 {
		t.Error("tok should be empty")
	}
	if !hasErr(err, "provide secret key") {
		t.Error("want secret key err, got:", err)
	}

	tok, err = generateToken("name", "key")
	if err != nil {
		t.Error(err)
	}
	if len(tok) == 0 {
		t.Error("tok should not be empty")
	}
}

func TestLoadConfigFromReader(t *testing.T) {
	t.Parallel()

	key, err := loadKeyFromConfig(strings.NewReader(testConfig))
	if err != nil {
		t.Error(err)
	}
	if key != "key" {
		t.Error("got:", key)
	}

	key, err = loadKeyFromConfig(strings.NewReader(testBadConfig))
	if !hasErr(err, "must set secret_key") {
		t.Error("wanted specific err, got:", err)
	}
}

func TestConfigPath(t *testing.T) {
	t.Parallel()

	w, err := os.Getwd()
	if err != nil {
		t.Error(err)
	}

	exp := filepath.Join(w, configFile)

	if got := configPath(); got != exp {
		t.Errorf("want: %q, got: %q", exp, got)
	}
}

func hasErr(err error, match string) bool {
	return strings.Contains(err.Error(), match)
}
