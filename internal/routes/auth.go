package routes

import (
	"fmt"
	"lazarus/internal/utils"
	"net/url"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type cookie struct {
	Name     string
	Value    string
	TTL      time.Duration
	HttpOnly bool
}

func (s *Server) authLogin(c *fiber.Ctx) error {
	provider := strings.ToLower(c.Params("provider"))
	if !s.srvAuth.IsAllowedProvider(provider) {
		return fiber.NewError(fiber.StatusBadRequest, "unsupported provider")
	}

	next := c.Query("next", "/")
	if !strings.HasPrefix(next, "/") {
		next = "/"
	}

	//state := utils.RandB64URL(32)
	verifier := utils.PKCEVerifier()
	challenge := utils.PKCEChallenge(verifier)

	// cookies: verifier + state + next
	// s.setCookie(c, cookie{Name: "sb_oauth_state", Value: state, TTL: 10 * time.Minute, HttpOnly: true})
	s.setCookie(c, cookie{Name: "sb_pkce_verifier", Value: verifier, TTL: 10 * time.Minute, HttpOnly: true})
	s.setCookie(c, cookie{Name: "sb_oauth_next", Value: next, TTL: 10 * time.Minute, HttpOnly: true})

	// Supabase social login endpoint
	// Note: code_challenge_method should be "s256" (lowercase) in Supabase Auth.
	authURL := fmt.Sprintf("%s/auth/v1/authorize", s.conf.AuthConfig.SupabaseURL)
	u, _ := url.Parse(authURL)
	q := u.Query()
	q.Set("provider", provider)
	q.Set("redirect_to", s.conf.AuthConfig.CallbackURL)

	//q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "s256")
	u.RawQuery = q.Encode()

	return c.Redirect(u.String(), fiber.StatusFound)
}

func (s *Server) authCallback(c *fiber.Ctx) error {
	code := c.Query("code")

	if code == "" {
		return c.Redirect(s.conf.AuthConfig.FrontendURL+"/auth/error", fiber.StatusFound)
	}

	// stateCookie := c.Cookies("sb_oauth_state")
	verifier := c.Cookies("sb_pkce_verifier")
	next := c.Cookies("sb_oauth_next")
	if next == "" || !strings.HasPrefix(next, "/") {
		next = "/"
	}

	if verifier == "" {
		s.clearAuthCookies(c)
		return c.Redirect(s.conf.AuthConfig.FrontendURL+"/auth/error", fiber.StatusFound)
	}

	sess, err := s.srvAuth.ExchangeCodeForSession(c.Context(), code, verifier)
	if err != nil {
		s.clearAuthCookies(c)
		return c.Redirect(s.conf.AuthConfig.FrontendURL+"/auth/error", fiber.StatusFound)
	}

	// store Supabase session tokens in httpOnly cookies
	// Weak point: cookie-stored refresh token is a crown jewel. Keep Secure+HttpOnly and consider rotating + short TTL later.
	s.setCookie(c, cookie{
		Name:     "sb_access_token",
		Value:    sess.AccessToken,
		TTL:      time.Duration(sess.ExpiresIn) * time.Second,
		HttpOnly: true,
	})
	s.setCookie(c, cookie{
		Name:     "sb_refresh_token",
		Value:    sess.RefreshToken,
		TTL:      90 * 24 * time.Hour, // your choice; Supabase refresh tokens can live long
		HttpOnly: true,
	})

	// cleanup pkce cookies
	// s.setCookie(c, cookie{Name: "sb_oauth_state", Value: "", TTL: -1, HttpOnly: true})
	s.setCookie(c, cookie{Name: "sb_pkce_verifier", Value: "", TTL: -1, HttpOnly: true})
	s.setCookie(c, cookie{Name: "sb_oauth_next", Value: "", TTL: -1, HttpOnly: true})

	return c.Redirect(s.conf.AuthConfig.FrontendURL+next, fiber.StatusFound)
}

func (s *Server) authLogout(ctx *fiber.Ctx) error {
	s.clearAuthCookies(ctx)
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (s *Server) clearAuthCookies(c *fiber.Ctx) {
	names := []string{
		"sb_access_token",
		"sb_refresh_token",
		"sb_oauth_state",
		"sb_pkce_verifier",
		"sb_oauth_next",
	}
	for _, n := range names {
		s.setCookie(c, cookie{Name: n, Value: "", TTL: -1, HttpOnly: true})
	}
}

func (s *Server) setCookie(c *fiber.Ctx, ck cookie) {
	exp := time.Now().Add(ck.TTL)
	if ck.TTL < 0 {
		exp = time.Unix(0, 0)
	}

	sameSite := fiber.CookieSameSiteLaxMode
	switch s.conf.AuthConfig.CookieSameSite {
	case "strict":
		sameSite = fiber.CookieSameSiteStrictMode
	case "none":
		sameSite = fiber.CookieSameSiteNoneMode
	default:
		sameSite = fiber.CookieSameSiteLaxMode
	}

	c.Cookie(&fiber.Cookie{
		Name:     ck.Name,
		Value:    ck.Value,
		Domain:   "", //   s.conf.AuthConfig.CookieDomain,
		Path:     "/",
		Expires:  exp,
		Secure:   s.conf.AuthConfig.CookieSecure,
		HTTPOnly: ck.HttpOnly,
		SameSite: sameSite,
	})
}
