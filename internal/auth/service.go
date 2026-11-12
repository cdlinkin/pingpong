package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type repository interface {
	CreateUser(ctx context.Context, login, email, passwordHash string) (int, error)
	GetUserByEmail(ctx context.Context, email string) (UserModel, error)
	GetUserByID(ctx context.Context, id int) (UserModel, error)

	SaveRefreshToken(ctx context.Context, userID int, hash string, expiresAt time.Time) error
	ConsumeRefreshToken(ctx context.Context, hash string) (int, error)
	DeleteRefreshToken(ctx context.Context, hash string) error
}

type Service struct {
	repository repository
	jwtSecret  []byte
}

func NewService(
	repository repository,
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
) (TokenPair, error) {
	email, err := validateEmail(email)
	if err != nil {
		return TokenPair{}, fmt.Errorf(
			"validate 'email': %w",
			err,
		)
	}

	userModel, err := s.repository.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return TokenPair{}, ErrInvalidCredentials
		}
		return TokenPair{}, fmt.Errorf(
			"get user by email: %w",
			err,
		)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userModel.PasswordHash), []byte(password)); err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}

	return s.issueTokens(ctx, userModel.ID)
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

func (s *Service) Refresh(ctx context.Context, refresh string) (TokenPair, error) {
	if refresh == "" {
		return TokenPair{}, ErrInvalidRefreshToken
	}

	userID, err := s.repository.ConsumeRefreshToken(ctx, hashToken(refresh))
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			return TokenPair{}, ErrInvalidRefreshToken
		}
		return TokenPair{}, fmt.Errorf(
			"consume refresh token: %w",
			err,
		)
	}

	return s.issueTokens(ctx, userID)
}

func (s *Service) Logout(ctx context.Context, refresh string) error {
	if refresh == "" {
		return nil
	}

	if err := s.repository.DeleteRefreshToken(ctx, hashToken(refresh)); err != nil {
		return fmt.Errorf(
			"delete refresh token: %w",
			err,
		)
	}

	return nil
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

func (s *Service) issueTokens(ctx context.Context, userID int) (TokenPair, error) {
	access, err := createToken(userID, s.jwtSecret)
	if err != nil {
		return TokenPair{}, fmt.Errorf(
			"create access token: %w",
			err,
		)
	}

	refresh, err := newRefreshToken()
	if err != nil {
		return TokenPair{}, err
	}

	if err := s.repository.SaveRefreshToken(ctx, userID, hashToken(refresh), time.Now().Add(refreshTTL)); err != nil {
		return TokenPair{}, fmt.Errorf(
			"save refresh token: %w",
			err,
		)
	}

	return TokenPair{
		Access:  access,
		Refresh: refresh,
	}, nil
}
