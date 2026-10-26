package acme

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/go-acme/lego/v5/challenge/http01"
	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/observability/logs"
	"github.com/traefik/traefik/v3/pkg/plugins/host"
)

// ChallengeHTTP HTTP challenge provider implements challenge.Provider.
// The challenges are registered in Traefik (see pkg/plugins/host): the ACME resolvers are plugins,
// and the router of any of them on an entry point answers the challenges of all of them.
type ChallengeHTTP struct{}

// NewChallengeHTTP creates a new ChallengeHTTP.
func NewChallengeHTTP() *ChallengeHTTP {
	return &ChallengeHTTP{}
}

// Present presents a challenge to obtain new ACME certificate.
func (c *ChallengeHTTP) Present(ctx context.Context, domain, token, keyAuth string) error {
	host.Register(challengeKey(token, domain), keyAuth)

	return nil
}

// CleanUp cleans the challenges when certificate is obtained.
func (c *ChallengeHTTP) CleanUp(ctx context.Context, domain, token, _ string) error {
	host.Unregister(challengeKey(token, domain))

	return nil
}

// Timeout calculates the maximum of time allowed to resolved an ACME challenge.
func (c *ChallengeHTTP) Timeout() (timeout, interval time.Duration) {
	return 60 * time.Second, 5 * time.Second
}

func (c *ChallengeHTTP) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	logger := log.Ctx(req.Context()).With().Str(logs.ProviderName, "acme").Logger()

	token, err := getPathParam(req.URL)
	if err != nil {
		logger.Error().Err(err).Msg("Unable to get token")
		rw.WriteHeader(http.StatusNotFound)
		return
	}

	if token != "" {
		domain, _, err := net.SplitHostPort(req.Host)
		if err != nil {
			logger.Debug().Err(err).Msg("Unable to split host and port. Fallback to request host.")
			domain = req.Host
		}

		tokenValue := c.getTokenValue(logger.WithContext(req.Context()), token, domain)
		if len(tokenValue) > 0 {
			rw.WriteHeader(http.StatusOK)
			_, err = rw.Write(tokenValue)
			if err != nil {
				logger.Error().Err(err).Msg("Unable to write token")
			}
			return
		}
	}

	rw.WriteHeader(http.StatusNotFound)
}

func (c *ChallengeHTTP) getTokenValue(ctx context.Context, token, domain string) []byte {
	logger := log.Ctx(ctx)
	logger.Debug().Msgf("Retrieving the ACME challenge for %s (token %q)...", domain, token)

	value, ok := host.Lookup(challengeKey(token, domain))
	if !ok {
		logger.Error().Msgf("Cannot retrieve the ACME challenge for %s (token %q)", domain, token)
		return nil
	}

	keyAuth, _ := value.(string)

	return []byte(keyAuth)
}

func challengeKey(token, domain string) string {
	return "acme-http-challenge/" + token + "/" + domain
}

func getPathParam(uri *url.URL) (string, error) {
	exp := regexp.MustCompile(fmt.Sprintf(`^%s([^/]+)/?$`, http01.PathPrefix))
	parts := exp.FindStringSubmatch(uri.Path)

	if len(parts) != 2 {
		return "", errors.New("missing token")
	}

	return parts[1], nil
}
