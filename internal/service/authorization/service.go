package authorization

import (
	"context"
	"errors"
	"lazarus/internal/config"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"lazarus/internal/repository"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type Service struct {
	ctx context.Context
	cfg *config.AppConfig
	log logger.AppLogger

	supportedProviders map[string]struct{}
	repo               *repository.Repo
	jwtKey             []byte
}

func NewService(ctx context.Context, log logger.AppLogger, cfg *config.AppConfig, repo *repository.Repo) *Service {
	return &Service{
		ctx:  ctx,
		cfg:  cfg,
		log:  log,
		repo: repo,

		supportedProviders: map[string]struct{}{
			"google": {},
			"github": {},
		},
		jwtKey: []byte(cfg.JWTKey),
	}
}

func (s *Service) IsAllowedProvider(p string) bool {
	_, ok := s.supportedProviders[strings.TrimSpace(p)]
	return ok
}

func (s *Service) GetTokenValidUntil() int64 {
	return time.Now().Add(time.Minute * time.Duration(s.cfg.JWTLive)).Unix()
}

func (s *Service) GetCodeChallenge(ctx context.Context, key uuid.UUID) (string, error) {
	return s.repo.GetKey(ctx, key)
}

func (s *Service) SetCodeChallenge(ctx context.Context, key string) (uuid.UUID, error) {
	return s.repo.SetKey(ctx, key)
}

func (s *Service) GoogleLogin(ctx context.Context, googleUser *entities.GoogleUser) (string, error) {
	lg := s.log.With(
		logger.WithString("provider", "google"),
		logger.WithEmail(googleUser.Email),
		logger.WithUserName(googleUser.Name),
	)
	usr, err := s.repo.GetUserByMail(ctx, googleUser.Email)
	if err != nil {
		lg.Error("error load user by mail", err)
		return "", errors.New("error load user by mail")
	}
	if usr == nil {
		if err = s.repo.AddGoogleUser(ctx, googleUser); err != nil {
			lg.Error("error add user", err)
			return "", errors.New("error add user")
		}
		usr, err = s.repo.GetUserByMail(ctx, googleUser.Email)
		if err != nil {
			lg.Error("error load user by mail after creation", err)
			return "", errors.New("error load user by mail after creation")
		}
	}
	//user exist or created, generate jwt
	jwtKey, err := s.generateJWT(usr)
	if err != nil {
		lg.Error("error generate jwt", err)
	}
	return jwtKey, err
}

func (s *Service) generateJWT(usr *entities.User) (string, error) {
	at := jwt.NewWithClaims(jwt.SigningMethodHS512, entities.UserJWT{
		UserID: usr.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * time.Duration(s.cfg.JWTLive))),
		},
	})
	return at.SignedString(s.jwtKey)
}
