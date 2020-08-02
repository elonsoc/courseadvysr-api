package main

//lifted from bondkeepr 2020-08-02

import (
	"crypto/ed25519"
	"log"
	"time"

	"github.com/o1egl/paseto"
)

var publicKey, privateKey, _ = ed25519.GenerateKey(nil)

//ExpTime is the time it takes for the token to expire
var ExpTime = time.Now().Add(5 * time.Minute)

//GenerateKey generates a PASETO Key with regards to the username
func GenerateKey(username string) (string, error) {

	jsonToken := paseto.JSONToken{
		Expiration: ExpTime,
	}

	footer := "COURSEADVYSR"

	jsonToken.Set("user", username)

	jsonToken.Set("issueTime", time.Now().String())

	//Should we provide the argon2id hash with the jsonToken?

	token, err := paseto.NewV2().Sign(privateKey, jsonToken, footer)
	if err != nil {
		return "", err
	}
	return token, err
}

//VerifyKey verifies if the given token matches
func VerifyKey(token string) error {
	var newJSONToken paseto.JSONToken
	var newFooter string

	err := paseto.NewV2().Verify(token, publicKey, &newJSONToken, &newFooter)
	if err != nil {
		log.Println(err)
		return err
	}
	return err
}
