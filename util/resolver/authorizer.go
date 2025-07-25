package resolver

import (
	"context"
	"encoding/base64"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/containerd/containerd/v2/core/remotes/docker"
	"github.com/containerd/containerd/v2/core/remotes/docker/auth"
	remoteserrors "github.com/containerd/containerd/v2/core/remotes/errors"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/moby/buildkit/session"
	sessionauth "github.com/moby/buildkit/session/auth"
	"github.com/moby/buildkit/util/bklog"
	"github.com/moby/buildkit/util/errutil"
	"github.com/moby/buildkit/util/flightcontrol"
	"github.com/moby/buildkit/version"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const defaultExpiration = 60

type authHandlerNS struct {
	counter int64 // needs to be 64bit aligned for 32bit systems

	handlers   map[string]*authHandler
	muHandlers sync.Mutex
	hosts      map[string][]docker.RegistryHost
	muHosts    sync.Mutex
	sm         *session.Manager
	g          flightcontrol.Group[[]docker.RegistryHost]
}

func newAuthHandlerNS(sm *session.Manager) *authHandlerNS {
	return &authHandlerNS{
		handlers: map[string]*authHandler{},
		hosts:    map[string][]docker.RegistryHost{},
		sm:       sm,
	}
}

func (a *authHandlerNS) get(ctx context.Context, host string, sm *session.Manager, g session.Group) *authHandler {
	if g != nil {
		if iter := g.SessionIterator(); iter != nil {
			for {
				id := iter.NextSession()
				if id == "" {
					break
				}
				h, ok := a.handlers[host+"/"+id]
				if ok {
					h.lastUsed = time.Now()
					return h
				}
			}
		}
	}

	// link another handler
	for k, h := range a.handlers {
		parts := strings.SplitN(k, "/", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[0] == host {
			if h.authority != nil {
				sessionID, ok, err := sessionauth.VerifyTokenAuthority(ctx, host, h.authority, sm, g)
				if err == nil && ok {
					a.handlers[host+"/"+sessionID] = h
					h.lastUsed = time.Now()
					return h
				}
			} else {
				sessionID, username, password, err := sessionauth.CredentialsFunc(sm, g)(host)
				if err == nil {
					if username == h.common.Username && password == h.common.Secret {
						a.handlers[host+"/"+sessionID] = h
						h.lastUsed = time.Now()
						return h
					}
				}
			}
		}
	}

	return nil
}

func (a *authHandlerNS) set(host, session string, h *authHandler) {
	a.handlers[host+"/"+session] = h
}

func (a *authHandlerNS) delete(h *authHandler) {
	maps.DeleteFunc(a.handlers, func(_ string, v *authHandler) bool {
		return v == h
	})
}

type dockerAuthorizer struct {
	client *http.Client

	sm       *session.Manager
	session  session.Group
	handlers *authHandlerNS
}

func newDockerAuthorizer(client *http.Client, handlers *authHandlerNS, sm *session.Manager, group session.Group) *dockerAuthorizer {
	return &dockerAuthorizer{
		client:   client,
		handlers: handlers,
		sm:       sm,
		session:  group,
	}
}

// Authorize handles auth request.
func (a *dockerAuthorizer) Authorize(ctx context.Context, req *http.Request) error {
	a.handlers.muHandlers.Lock()
	defer a.handlers.muHandlers.Unlock()

	// skip if there is no auth handler
	ah := a.handlers.get(ctx, req.URL.Host, a.sm, a.session)
	if ah == nil {
		return nil
	}

	authHeader, err := ah.authorize(ctx, a.sm, a.session)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", authHeader)
	return nil
}

func (a *dockerAuthorizer) getCredentials(host string) (sessionID, username, secret string, err error) {
	return sessionauth.CredentialsFunc(a.sm, a.session)(host)
}

func (a *dockerAuthorizer) AddResponses(ctx context.Context, responses []*http.Response) error {
	a.handlers.muHandlers.Lock()
	defer a.handlers.muHandlers.Unlock()

	last := responses[len(responses)-1]
	host := last.Request.URL.Host

	handler := a.handlers.get(ctx, host, a.sm, a.session)

	for _, c := range auth.ParseAuthHeader(last.Header) {
		switch c.Scheme {
		case auth.BearerAuth:
			var oldScopes []string
			if err := invalidAuthorization(c, responses); err != nil {
				a.handlers.delete(handler)

				if handler != nil {
					oldScopes = handler.common.Scopes
				}
				handler = nil

				// this hacky way seems to be the best method to detect that error is fatal and should not be retried with a new token
				if c.Parameters["error"] == "insufficient_scope" && parseScopes(oldScopes).contains(parseScopes(strings.Split(c.Parameters["scope"], " "))) {
					return err
				}
			}

			// reuse existing handler
			//
			// assume that one registry will return the common
			// challenge information, including realm and service.
			// and the resource scope is only different part
			// which can be provided by each request.
			if handler != nil {
				return nil
			}

			var username, secret string
			sessionID, pubKey, err := sessionauth.GetTokenAuthority(ctx, host, a.sm, a.session)
			if err != nil {
				return err
			}
			if pubKey == nil {
				sessionID, username, secret, err = a.getCredentials(host)
				if err != nil {
					return err
				}
			}

			common, err := auth.GenerateTokenOptions(ctx, host, username, secret, c)
			if err != nil {
				return err
			}
			common.Scopes = parseScopes(append(common.Scopes, oldScopes...)).normalize()

			a.handlers.set(host, sessionID, newAuthHandler(host, a.client, c.Scheme, pubKey, common))

			return nil
		case auth.BasicAuth:
			sessionID, username, secret, err := a.getCredentials(host)
			if err != nil {
				return err
			}

			if username != "" && secret != "" {
				a.handlers.set(host, sessionID, newAuthHandler(host, a.client, c.Scheme, nil, auth.TokenOptions{
					Username: username,
					Secret:   secret,
				}))
				return nil
			}
		}
	}
	return errors.Wrap(cerrdefs.ErrNotImplemented, "failed to find supported auth scheme")
}

// authResult is used to control limit rate.
type authResult struct {
	token   string
	expires time.Time
}

// authHandler is used to handle auth request per registry server.
type authHandler struct {
	g flightcontrol.Group[*authResult]

	client *http.Client

	// only support basic and bearer schemes
	scheme auth.AuthenticationScheme

	// common contains common challenge answer
	common auth.TokenOptions

	// scopedTokens caches token indexed by scopes, which used in
	// bearer auth case
	scopedTokens   map[string]*authResult
	scopedTokensMu sync.Mutex

	lastUsed time.Time

	host string

	authority *[32]byte
}

func newAuthHandler(host string, client *http.Client, scheme auth.AuthenticationScheme, authority *[32]byte, opts auth.TokenOptions) *authHandler {
	return &authHandler{
		host:         host,
		client:       client,
		scheme:       scheme,
		common:       opts,
		scopedTokens: map[string]*authResult{},
		lastUsed:     time.Now(),
		authority:    authority,
	}
}

func (ah *authHandler) authorize(ctx context.Context, sm *session.Manager, g session.Group) (string, error) {
	switch ah.scheme {
	case auth.BasicAuth:
		return ah.doBasicAuth()
	case auth.BearerAuth:
		return ah.doBearerAuth(ctx, sm, g)
	default:
		return "", errors.Wrapf(cerrdefs.ErrNotImplemented, "failed to find supported auth scheme: %s", string(ah.scheme))
	}
}

func (ah *authHandler) doBasicAuth() (string, error) {
	username, secret := ah.common.Username, ah.common.Secret

	if username == "" || secret == "" {
		return "", errors.New("failed to handle basic auth because missing username or secret")
	}

	authHeader := base64.StdEncoding.EncodeToString([]byte(username + ":" + secret))
	return fmt.Sprintf("Basic %s", authHeader), nil
}

func (ah *authHandler) doBearerAuth(ctx context.Context, sm *session.Manager, g session.Group) (token string, err error) {
	// copy common tokenOptions
	to := ah.common

	to.Scopes = parseScopes(docker.GetTokenScopes(ctx, to.Scopes)).normalize()

	// Docs: https://distribution.github.io/distribution/spec/auth/scope
	scoped := strings.Join(to.Scopes, " ")

	res, err := ah.g.Do(ctx, scoped, func(ctx context.Context) (*authResult, error) {
		ah.scopedTokensMu.Lock()
		r, exist := ah.scopedTokens[scoped]
		ah.scopedTokensMu.Unlock()
		if exist {
			if r.expires.IsZero() || r.expires.After(time.Now()) {
				return r, nil
			}
		}
		r, err := ah.fetchToken(ctx, sm, g, to)
		if err != nil {
			return nil, err
		}
		ah.scopedTokensMu.Lock()
		ah.scopedTokens[scoped] = r
		ah.scopedTokensMu.Unlock()
		return r, nil
	})
	if err != nil || res == nil {
		return "", err
	}
	return res.token, nil
}

func (ah *authHandler) fetchToken(ctx context.Context, sm *session.Manager, g session.Group, to auth.TokenOptions) (r *authResult, err error) {
	var issuedAt time.Time
	var expires int
	var token string
	defer func() {
		token = fmt.Sprintf("Bearer %s", token)

		if err == nil {
			r = &authResult{token: token}
			if issuedAt.IsZero() {
				issuedAt = time.Now()
			}
			if exp := issuedAt.Add(time.Duration(float64(expires)*0.9) * time.Second); time.Now().Before(exp) {
				r.expires = exp
			}
		}
	}()

	if ah.authority != nil {
		resp, err := sessionauth.FetchToken(ctx, &sessionauth.FetchTokenRequest{
			ClientID: "buildkit-client",
			Host:     ah.host,
			Realm:    to.Realm,
			Service:  to.Service,
			Scopes:   to.Scopes,
		}, sm, g)
		if err != nil {
			return nil, err
		}
		if resp.ExpiresIn == 0 {
			resp.ExpiresIn = defaultExpiration
		}
		expires = int(resp.ExpiresIn)
		// We later check issuedAt.isZero, which would return
		// false if converted from zero Unix time. Therefore,
		// zero time value in response is handled separately
		if resp.IssuedAt == 0 {
			issuedAt = time.Time{}
		} else {
			issuedAt = time.Unix(resp.IssuedAt, 0)
		}
		token = resp.Token
		return nil, nil
	}

	hdr := http.Header{}
	hdr.Set("User-Agent", version.UserAgent())

	// fetch token for the resource scope
	if to.Secret != "" {
		defer func() {
			err = errors.Wrap(errutil.WithDetails(err), "failed to fetch oauth token")
		}()
		// try GET first because Docker Hub does not support POST
		// switch once support has landed
		resp, err := auth.FetchToken(ctx, ah.client, nil, to)
		if err != nil {
			var errStatus remoteserrors.ErrUnexpectedStatus
			if errors.As(err, &errStatus) {
				// retry with POST request
				// As of September 2017, GCR is known to return 404.
				// As of February 2018, JFrog Artifactory is known to return 401.
				if (errStatus.StatusCode == http.StatusMethodNotAllowed && to.Username != "") || errStatus.StatusCode == http.StatusNotFound || errStatus.StatusCode == http.StatusUnauthorized {
					resp, err := auth.FetchTokenWithOAuth(ctx, ah.client, hdr, "buildkit-client", to)
					if err != nil {
						return nil, err
					}
					if resp.ExpiresInSeconds == 0 {
						resp.ExpiresInSeconds = defaultExpiration
					}
					issuedAt, expires = resp.IssuedAt, resp.ExpiresInSeconds
					token = resp.AccessToken
					return nil, nil
				}
				bklog.G(ctx).WithFields(logrus.Fields{
					"status": errStatus.Status,
					"body":   string(errStatus.Body),
				}).Debugf("token request failed")
			}
			return nil, err
		}
		if resp.ExpiresInSeconds == 0 {
			resp.ExpiresInSeconds = defaultExpiration
		}
		issuedAt, expires = resp.IssuedAt, resp.ExpiresInSeconds
		token = resp.Token
		return nil, nil
	}
	// do request anonymously
	resp, err := auth.FetchToken(ctx, ah.client, hdr, to)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch anonymous token")
	}
	if resp.ExpiresInSeconds == 0 {
		resp.ExpiresInSeconds = defaultExpiration
	}
	issuedAt, expires = resp.IssuedAt, resp.ExpiresInSeconds

	token = resp.Token
	return nil, nil
}

func invalidAuthorization(c auth.Challenge, responses []*http.Response) error {
	errStr := c.Parameters["error"]
	if errStr == "" {
		return nil
	}

	n := len(responses)
	if n == 1 || (n > 1 && !sameRequest(responses[n-2].Request, responses[n-1].Request)) {
		return nil
	}

	return errors.Wrapf(docker.ErrInvalidAuthorization, "server message: %s", errStr)
}

func sameRequest(r1, r2 *http.Request) bool {
	if r1.Method != r2.Method {
		return false
	}
	if *r1.URL != *r2.URL {
		return false
	}
	return true
}

type scopes map[string]map[string]struct{}

func parseScopes(s []string) scopes {
	// https://distribution.github.io/distribution/spec/auth/scope/
	m := map[string]map[string]struct{}{}
	for _, scopeStr := range s {
		if scopeStr == "" {
			return nil
		}
		// The scopeStr may have strings that contain multiple scopes separated by a space.
		for _, scope := range strings.Split(scopeStr, " ") {
			parts := strings.SplitN(scope, ":", 3)
			names := []string{parts[0]}
			if len(parts) > 1 {
				names = append(names, parts[1])
			}
			var actions []string
			if len(parts) == 3 {
				actions = append(actions, strings.Split(parts[2], ",")...)
			}
			name := strings.Join(names, ":")
			ma, ok := m[name]
			if !ok {
				ma = map[string]struct{}{}
				m[name] = ma
			}

			for _, a := range actions {
				ma[a] = struct{}{}
			}
		}
	}
	return m
}

func (s scopes) normalize() []string {
	names := make([]string, 0, len(s))
	for n := range s {
		names = append(names, n)
	}
	slices.Sort(names)

	out := make([]string, 0, len(s))

	for _, n := range names {
		actions := make([]string, 0, len(s[n]))
		for a := range s[n] {
			actions = append(actions, a)
		}
		slices.Sort(actions)

		out = append(out, n+":"+strings.Join(actions, ","))
	}
	return out
}

func (s scopes) contains(s2 scopes) bool {
	for n := range s2 {
		v, ok := s[n]
		if !ok {
			return false
		}
		for a := range s2[n] {
			if _, ok := v[a]; !ok {
				return false
			}
		}
	}
	return true
}
