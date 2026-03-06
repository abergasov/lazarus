package routes

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"lazarus/internal/entities"
	"lazarus/internal/utils"
	"net/http"
	"net/url"
	"strings"

	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	RefreshCookie     = "rc"
	TokenCookie       = "tc"
	UserIDCookie      = "u"
	oauthGoogleUrlAPI = "https://www.googleapis.com/oauth2/v2/userinfo?access_token="
)

func (s *Server) oauthGoogleLogin(c *fiber.Ctx) error {
	// generate cookie
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	c.Cookie(&fiber.Cookie{
		Name:    "oauthstate",
		Value:   state,
		Expires: time.Now().Add(365 * 24 * time.Hour),
	})
	// Build redirect URL dynamically from the incoming request host so the app
	// works from localhost, LAN IP, or any other hostname without config changes.
	scheme := "http"
	if c.Protocol() == "https" {
		scheme = "https"
	}
	cfg := *s.googleOAuth // shallow copy so we don't mutate the shared config
	cfg.RedirectURL = scheme + "://" + c.Get("Host") + "/api/auth/google/callback"
	return c.Redirect(cfg.AuthCodeURL(state), http.StatusTemporaryRedirect)
}

func (s *Server) oauthGoogleCallback(c *fiber.Ctx) error {
	// Read oauthState from Cookie
	oauthState := c.Cookies("oauthstate", "")
	if c.FormValue("state") != oauthState {
		s.log.Error("invalid oauth google state", errors.New("form and cookie mismatch"))
		return c.Redirect("/", http.StatusTemporaryRedirect)
	}

	usr, err := s.getUserDataFromGoogle(c.FormValue("code"))
	if err != nil {
		s.log.Error("error get data from google", fmt.Errorf("error get user data from google: %w", err))
		return c.Redirect("/", http.StatusTemporaryRedirect)
	}

	jwt, err := s.srvAuth.GoogleLogin(c.UserContext(), usr)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "error login with google"})
	}
	code, err := s.srvAuth.SetCodeChallenge(c.UserContext(), jwt)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": "error set code challenge"})
	}
	s.setSecretCookie(c, TokenCookie, jwt)
	frontendURL := strings.TrimSuffix(s.conf.FrontendURL, "/")
	redirectURL := frontendURL + "/?" + url.Values{"code": {code.String()}}.Encode()
	return c.Redirect(redirectURL, http.StatusTemporaryRedirect)
}

func (s *Server) getUserDataFromGoogle(code string) (*entities.GoogleUser, error) {
	// Use code to get token and get user info from Google.
	token, err := s.googleOAuth.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("code exchange wrong: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	res, respCode, err := utils.GetCurl[entities.GoogleUser](ctx, oauthGoogleUrlAPI+token.AccessToken, nil)
	if err != nil || respCode != http.StatusOK {
		return nil, fmt.Errorf("failed getting user info code %d: %w", respCode, err)
	}
	return res, nil
}

func (s *Server) setSecretCookie(c *fiber.Ctx, keyName, keyValue string) {
	exp := time.Unix(s.srvAuth.GetTokenValidUntil(), 0)
	maxAge := 0

	if keyValue == "" {
		exp = time.Now().Add(-24 * time.Hour)
		maxAge = -1
	}

	c.Cookie(&fiber.Cookie{
		Name:     keyName,
		Value:    keyValue,
		Path:     "/",              // critical
		Domain:   "",               // local
		HTTPOnly: true,             // critical
		Secure:   s.conf.SSLEnable, // local
		SameSite: fiber.CookieSameSiteLaxMode,
		Expires:  exp,
		MaxAge:   maxAge,
	})
}

func (s *Server) exchangeCode(c *fiber.Ctx) error {
	var u struct {
		Code uuid.UUID `json:"code"`
	}
	if err := json.Unmarshal(c.Body(), &u); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{})
	}
	jwt, err := s.srvAuth.GetCodeChallenge(c.UserContext(), u.Code)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{})
	}
	// Set the cookie here too — belt-and-suspenders in case the redirect cookie was lost
	s.setSecretCookie(c, TokenCookie, jwt)
	return c.JSON(map[string]interface{}{"ok": true})
}

func (s *Server) Logout(c *fiber.Ctx) error {
	s.setSecretCookie(c, TokenCookie, "")
	s.setSecretCookie(c, RefreshCookie, "")
	s.setSecretCookie(c, UserIDCookie, "")
	return c.Redirect("/", fiber.StatusSeeOther) // 303
}
