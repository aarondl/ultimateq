package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aarondl/ultimateq/config"
	"github.com/dgrijalva/jwt-go"
)

const configFile = "config.toml"

var usage = `Usage: uqtokgen <extname>
Generate a token using the configured secret token`

func main() {
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}

	extName := os.Args[1]

	f, err := os.Open(configFile)
	if err != nil {
		fmt.Printf("failed to open file: %s\n", configPath())
		os.Exit(1)
	}
	defer f.Close()

	tok, err := run(extName, f)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(tok)
}

func run(name string, reader io.Reader) (tok string, err error) {
	var key string

	if key, err = loadKeyFromConfig(reader); err != nil {
		return tok, err
	}

	return generateToken(name, key)
}

func generateToken(name, key string) (string, error) {
	if len(name) == 0 {
		return "", errors.New("must provide an extension name to generate a token")
	}
	if len(key) == 0 {
		return "", errors.New("must provide secret key to sign the token")
	}

	tok := jwt.New(jwt.SigningMethodHS512)
	tok.Claims["uq"] = "extension"
	tok.Claims["ext"] = name

	return tok.SignedString([]byte(key))
}

func loadKeyFromConfig(reader io.Reader) (string, error) {
	cfg := config.New().FromReader(reader)
	if !cfg.Validate() {
		return "", errors.New(concatErrors(cfg.Errors()))
	}

	key, ok := cfg.SecretKey()
	if !ok || len(key) == 0 {
		return "", errors.New("must set secret_key in the configuration")
	}

	return key, nil
}

func configPath() string {
	path := configFile
	if p, err := filepath.Abs(configFile); err == nil {
		path = p
	}

	return path
}

func concatErrors(errs []error) string {
	buf := &bytes.Buffer{}
	for _, err := range errs {
		fmt.Fprintln(buf, err.Error())
	}

	return buf.String()
}
