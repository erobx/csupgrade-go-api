package api

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// insert new user, login a user, get user details (inventory, recents, stats, etc.),
// update user (update balance, insert new items, delete items, etc.), buy items

// Responsible for every user interaction, new, remove, updates
type UserService interface {
	New(user *NewUserRequest) (string, error)
	Login(request *NewLoginRequest) (User, Inventory, error)
	GetUser(userID string) (User, error)
	GetInventory(userID string) (Inventory, error)
	GetRecentTradeups(userID string) ([]RecentTradeup, error)
	GetRecentWinnings(userID string) ([]Item, error)
	GetStats(userID string) error
}

type UserRepository interface {
	CreateUser(*NewUserRequest) (string, error)
	GetUserByID(userID string) (User, error)
	GetUserAndHashByEmail(email string) (User, string, error)
	GetInventory(userID string) (Inventory, error)
	GetRecentTradeups(userID string) ([]RecentTradeup, error)
	GetRecentWinnings(userID string) ([]Item, error)
}

type userService struct {
	storage UserRepository
	logger LogService
}

// Handles all user requests
func NewUserService(userRepo UserRepository, logger LogService) UserService {
	return &userService{storage: userRepo, logger: logger}
}

// Creates a new user and returns their ID
func (u *userService) New(user *NewUserRequest) (string, error) {
	err := ValidateNewUserRequest(user)
	if err != nil {
		return "", err
	}

	// Check if user already exists
	_, _, err = u.storage.GetUserAndHashByEmail(user.Email)
	if err == nil {
		return "", errors.New("email already used")
	}

	// Normalization
	user.Email = strings.TrimSpace(user.Email)

	return u.storage.CreateUser(user)
}

// Logs in an existing user, gets their data and inventory
func (u *userService) Login(request *NewLoginRequest) (User, Inventory, error) {
	var user User
	var inv Inventory
	err := ValidateLoginRequest(request)
	if err != nil {
		return user, inv, err
	}

	request.Email = strings.TrimSpace(request.Email)

	user, hash, err := u.storage.GetUserAndHashByEmail(request.Email)
	if err != nil {
		return user, inv, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(request.Password))
	if err != nil {
		return user, inv, err
	}

	inv, err = u.storage.GetInventory(user.ID)
	if err != nil {
		return user, inv, err
	}

	return user, inv, nil
}

func (u *userService) GetUser(userID string) (User, error) {
	user, err := u.storage.GetUserByID(userID)
	if err != nil {
		return user, err
	}

	return user, nil
}

func (u *userService) GetInventory(userID string) (Inventory, error) {
	return u.storage.GetInventory(userID)
}

func (u *userService) GetRecentTradeups(userID string) ([]RecentTradeup, error) {
	return u.storage.GetRecentTradeups(userID)
}

func (u *userService) GetRecentWinnings(userID string) ([]Item, error) {
	return u.storage.GetRecentWinnings(userID)
}

func (u *userService) GetStats(userID string) error {
	return nil
}

/*
Validators
*/

func ValidateNewUserRequest(user *NewUserRequest) error {
	if user.Email == "" {
		return errors.New("email cannot be empty")
	}

	if user.Username == "" {
		return errors.New("username cannot be empty")
	}

	if user.Password == "" {
		return errors.New("password cannot be empty")
	}

	return nil
}

func ValidateLoginRequest(user *NewLoginRequest) error {
	if user.Email == "" {
		return errors.New("email cannot be empty")
	}

	if user.Password == "" {
		return errors.New("password cannot be empty")
	}
	
	return nil
}
