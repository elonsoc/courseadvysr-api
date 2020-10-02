package main

//lifted from bondkeepr 2020-08-02

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	//ErrInvalidHash lol
	ErrInvalidHash = errors.New("the encoded hash is not in the correct format")
	//ErrIncompatibleVersion lol
	ErrIncompatibleVersion = errors.New("incompatible version of argon2")
)

//Parameters for argon2id encryption scheme of passwords
type params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

//GeneratePasswordHash generates the hash of the password and sends it to the db
func GeneratePasswordHash(password string) string {
	p := &params{
		memory:      64 * 1024,
		iterations:  3,
		parallelism: 2,
		saltLength:  16,
		keyLength:   32,
	}

	encodedHash, err := genPass(password, p)
	if err != nil {
		log.Fatal(err)
	}

	//DEBUG: making sure hash is developed
	log.Println(encodedHash)

	//return header based on account successfully created
	return encodedHash
}

//CheckPasswordHash pulls the password of the specified user and compares.
func CheckPasswordHash(userPassword string, username string) (bool, error) {

	//passHash is pulled from the password hash in postgres
	var passHash, err = GetHash(username)
	if err != nil {
		//we should just return errors all the way up and show with UI
		return false, err
	}

	match, err := comparePasswordAndHash(userPassword, passHash)
	if err != nil {
		/**
			2020-10-02 Side note
			I'm making the change from logging and exiting to kinda ignoring it
			since I return "" for passHash in GetHash and leads to an error
			when the account is not properly valid. I shouldn't do this.

			TODO: better error handling, more centralized.
			log.Fatal(err)
		**/
		return false, err
	}

	//return header based on if password matches or not
	return match, nil
}

func genPass(password string, p *params) (encodedHash string, err error) {
	salt, err := genBytes(p.saltLength)
	if err != nil {
		return " ", err
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		p.iterations,
		p.memory,
		p.parallelism,
		p.keyLength,
	)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encodedHash = fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.memory,
		p.iterations,
		p.parallelism,
		b64Salt,
		b64Hash,
	)

	return encodedHash, nil
}

func genBytes(n uint32) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func comparePasswordAndHash(password, encodedHash string) (match bool, err error) {
	p, salt, hash, err := decodeHash(encodedHash)
	if err != nil {
		return false, err
	}

	otherHash := argon2.IDKey([]byte(password), salt, p.iterations, p.memory, p.parallelism, p.keyLength)

	if subtle.ConstantTimeCompare(hash, otherHash) == 1 {
		return true, nil
	}

	return false, nil
}

func decodeHash(encodedHash string) (p *params, salt, hash []byte, err error) {
	vals := strings.Split(encodedHash, "$")
	if len(vals) != 6 {
		return nil, nil, nil, ErrInvalidHash
	}

	var version int
	_, err = fmt.Sscanf(vals[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, err
	}

	if version != argon2.Version {
		return nil, nil, nil, ErrIncompatibleVersion
	}

	p = &params{}

	_, err = fmt.Sscanf(vals[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism)

	if err != nil {
		return nil, nil, nil, err
	}

	salt, err = base64.RawStdEncoding.DecodeString(vals[4])
	if err != nil {
		return nil, nil, nil, err
	}
	p.saltLength = uint32(len(salt))

	hash, err = base64.RawStdEncoding.DecodeString(vals[5])
	if err != nil {
		return nil, nil, nil, err
	}
	p.keyLength = uint32(len(hash))

	return p, salt, hash, nil
}
