package authorization

import (
	"context"
	"errors"
	"lazarus/internal/config"
	"lazarus/internal/entities"
	"lazarus/internal/logger"
	"lazarus/internal/utils"
	"strings"
)

type Service struct {
	ctx context.Context
	cfg *config.AppConfig
	log logger.AppLogger

	supportedProviders map[string]struct{}
}

func NewService(ctx context.Context, cfg *config.AppConfig, log logger.AppLogger) *Service {
	return &Service{
		ctx: ctx,
		cfg: cfg,
		log: log,

		supportedProviders: map[string]struct{}{
			"google": {},
			"github": {},
		},
	}
}

func (s *Service) IsAllowedProvider(p string) bool {
	_, ok := s.supportedProviders[strings.TrimSpace(p)]
	return ok
}

func (s *Service) ExchangeCodeForSession(ctx context.Context, code, verifier string) (*entities.SupabaseAuthSession, error) {
	// Supabase PKCE code exchange happens at:
	// POST {SUPABASE_URL}/auth/v1/token?grant_type=pkce
	// form: auth_code=<code>&code_verifier=<verifier>
	//
	// The docs show exchanging "code" for session (pkce flow). :contentReference[oaicite:0]{index=0}
	endpoint := s.cfg.AuthConfig.SupabaseURL + "/auth/v1/token?grant_type=pkce"
	sess, resCode, err := utils.PostCurl[entities.SupabaseAuthSession](ctx, endpoint, map[string]string{
		"auth_code":     code,
		"code_verifier": verifier,
	}, map[string]string{
		"Content-Type":  "application/json",
		"apikey":        s.cfg.AuthConfig.SupabaseAnon,
		"Authorization": "Bearer " + s.cfg.AuthConfig.SupabaseAnon,
	})
	if err != nil {
		s.log.Error("error exchanging code for session", err, logger.WithHTTPCode(resCode))
		return nil, errors.New("error exchanging code for session")
	}
	if sess.AccessToken == "" || sess.RefreshToken == "" {
		return nil, errors.New("empty session tokens")
	}
	return sess, nil
}
