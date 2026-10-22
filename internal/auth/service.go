package auth

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repository *Repository
	jwtSecret  []byte
}

func NewService(
	repository *Repository,
	jwtSecret []byte,
) *Service {
	return &Service{
		repository: repository,
		jwtSecret:  jwtSecret,
	}
}

func (s *Service) Login(
	ctx context.Context,
	email, password string,
) (string, error) {
	email, err := validateEmail(email)
	if err != nil {
		return "", fmt.Errorf(
			"validate 'email': %w",
			err,
		)
	}

	userModel, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf(
			"get user by email: %w",
			err,
		)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userModel.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return createToken(userModel.ID, s.jwtSecret)
}

func (s *Service) Register(
	ctx context.Context,
	login, email, password string,
) (int, error) {
	email, err := validateEmail(email)
	if err != nil {
		return 0, fmt.Errorf(
			"validate 'email': %w",
			err,
		)
	}

	if err := validateLogin(login); err != nil {
		return 0, fmt.Errorf(
			"validate 'login': %w",
			err,
		)
	}

	if err := validatePassword(password); err != nil {
		return 0, fmt.Errorf(
			"validate 'password': %w",
			err,
		)
	}

	passwordHash, err := hashPassword(password)
	if err != nil {
		return 0, fmt.Errorf(
			"hash password: %w",
			err,
		)
	}

	userID, err := s.repository.CreateUser(ctx, login, email, passwordHash)
	if err != nil {
		return 0, fmt.Errorf(
			"create user: %w",
			err,
		)
	}

	return userID, nil
}

func (s *Service) Me(ctx context.Context, userID int) (UserModel, error) {
	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		return UserModel{}, fmt.Errorf(
			"get user by id: %w",
			err,
		)
	}

	return user, nil
}

func hashPassword(password string) (string, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(passwordHash), nil
}
