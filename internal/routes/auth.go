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

// externalScheme returns the scheme the client originally used, respecting
// X-Forwarded-Proto set by reverse proxies like ngrok or load balancers.
func externalScheme(c *fiber.Ctx) string {
	if fp := c.Get("X-Forwarded-Proto"); fp != "" {
		return fp
	}
	return c.Protocol()
}

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
	scheme := externalScheme(c)
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

	// Build redirect URL dynamically to match what was sent in the login request
	scheme := externalScheme(c)
	callbackURL := scheme + "://" + c.Get("Host") + "/api/auth/google/callback"

	usr, err := s.getUserDataFromGoogle(c.FormValue("code"), callbackURL)
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
	// Redirect to the same host the user came from
	frontendBase := scheme + "://" + c.Get("Host")
	redirectTo := frontendBase + "/?" + url.Values{"code": {code.String()}}.Encode()
	return c.Redirect(redirectTo, http.StatusTemporaryRedirect)
}

func (s *Server) getUserDataFromGoogle(code string, redirectURL string) (*entities.GoogleUser, error) {
	// Use code to get token and get user info from Google.
	cfg := *s.googleOAuth
	cfg.RedirectURL = redirectURL
	token, err := cfg.Exchange(context.Background(), code)
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
		Secure:   c.Protocol() == "https",
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
