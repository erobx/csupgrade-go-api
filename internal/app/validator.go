package app

import (
	"errors"
	"log"
)

type Validator interface {
	ValidateUserID(userID, jwtUserID string) error
}

type validator struct {

}

func NewValidator() Validator {
	return &validator{}
}

func (v *validator) ValidateUserID(userID, jwtUserID string) error {
	if userID != jwtUserID {
		log.Println("userID not the same as jwtID")
		log.Printf("%s - %s\n", userID, jwtUserID)
		return errors.New("provided userID does not match ID in jwt")
	}
	return nil
}
