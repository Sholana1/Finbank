package app

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sholana1/fintechdemo/identity/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user domain.User) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type RegisterUserInput struct {
	Email       string
	PhoneNumber string
	Password    string
}

type RegisterUser struct {
	users  UserRepository
	hasher PasswordHasher
	now    func() time.Time
}

func NewRegisterUser(users UserRepository, hasher PasswordHasher, now func() time.Time) *RegisterUser {
	return &RegisterUser{users: users, hasher: hasher, now: now}
}

func (uc *RegisterUser) Execute(ctx context.Context, in RegisterUserInput) (domain.User, error) {
	email, err := domain.NewEmail(in.Email)

	if err != nil {
		return domain.User{}, err
	}

	phone, err := domain.NewPhoneNumber(in.PhoneNumber)
	if err != nil {
		return domain.User{}, err
	}

	if err := domain.ValidatePassword(in.Password); err != nil {
		return domain.User{}, err
	}

	hash, err := uc.hasher.Hash(in.Password)
	if err != nil {
		return domain.User{}, fmt.Errorf("register user: hash password: %w", err)
	}

	id, err := uuid.NewV7()
	if err != nil {
		return domain.User{}, fmt.Errorf("register user: generate id: %w", err)
	}

	user := domain.NewUser(id.String(), email, phone, hash, uc.now())

	if err := uc.users.Create(ctx, user); err != nil {
		return domain.User{}, fmt.Errorf("register user: %w", err)
	}

	return user, nil
}
