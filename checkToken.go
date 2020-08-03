package main

//lifted from bondkeepr 2020-08-02

import (
	"log"
)

func CheckToken(token string) error {

	err := VerifyKey(token)
	if err != nil {
		log.Printf(err.Error())
		return err
	}

	return nil
}
