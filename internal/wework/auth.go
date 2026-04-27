package wework

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Auth0Config struct {
	ClientID    string `json:"client_id"`
	Domain      string `json:"domain"`
	RedirectURI string `json:"redirect_uri"`
	Audience    string `json:"audience"`
}

type WeWorkAuth struct {
	username      string
	password      string
	client        *BaseClient
	config        *Auth0Config
	codeVerifier  string
	codeChallenge string
}

type OAuthTokenResponse struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

type LoginByAuth0TokenResponse struct {
	Token            string `json:"token"`
	IDToken          string `json:"idToken"`
	UseRefreshTokens bool   `json:"useRefreshTokens"`
	RefreshToken     string `json:"refreshToken"`
	A0token          string `json:"a0token"`
	SessionID        string `json:"sessionId"`
	Username         string `json:"username"`
	A0rtoken         string `json:"a0rtoken"`
	A0Tokens         struct {
		AccessToken  string `json:"access_token"`
		Audience     string `json:"audience"`
		ClientID     string `json:"client_id"`
		ExpiresIn    int    `json:"expires_in"`
		IDToken      string `json:"id_token"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
		TokenType    string `json:"token_type"`
	} `json:"a0Tokens"`
	AccessToken string `json:"accessToken"`
}

type WeWorkLoginError struct {
	Message     string `json:"message"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Description string `json:"description"`
	StatusCode  int    `json:"statusCode"`
}

func (e *WeWorkLoginError) Error() string {
	return fmt.Sprintf("%s (%s)", e.Message, e.Code)
}

func (e *WeWorkLoginError) As(target any) bool {
	if p, ok := target.(*WeWorkLoginError); ok {
		*p = *e
		return true
	}
	return false
}

func (e *WeWorkLoginError) Is(target error) bool {
	t, ok := target.(*WeWorkLoginError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}
func NewWeWorkAuth(username, password string) (*WeWorkAuth, error) {
	baseClient, err := NewBaseClient()
	if err != nil {
		return nil, err
	}

	baseClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	auth := &WeWorkAuth{
		username: username,
		password: password,
		client:   baseClient,
	}

	if err := auth.getAuth0Config(); err != nil {
		return nil, err
	}

	auth.codeVerifier = generateCodeVerifier()
	auth.codeChallenge = generateCodeChallenge(auth.codeVerifier)

	return auth, nil
}

func (w *WeWorkAuth) getAuth0Config() error {
	baseURL := "https://members.wework.com/workplaceone/api/auth0/config"
	params := url.Values{}
	params.Add("companyId", "00000000-0000-0000-0000-000000000000")
	params.Add("domain", "members.wework.com")

	resp, err := w.client.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return fmt.Errorf("failed to get auth0 config: %w", err)
	}
	defer resp.Body.Close()

	var config Auth0Config
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return fmt.Errorf("failed to decode auth0 config: %w", err)
	}

	w.config = &config
	weWorkUrl, err := url.Parse("https://members.wework.com")
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}

	w.client.Jar.SetCookies(weWorkUrl, []*http.Cookie{
		{
			Name:  "auth0.zE51Ep7FttlmtQV6ZEGyJKsY2jD1EtAu.is.authenticated",
			Value: "true",
		},
		{
			Name:  "_legacy_auth0.zE51Ep7FttlmtQV6ZEGyJKsY2jD1EtAu.is.authenticated",
			Value: "true",
		},
	})

	return nil
}

func (w *WeWorkAuth) Authenticate() (*LoginByAuth0TokenResponse, *OAuthTokenResponse, error) {
	if res, tokens, ok, err := w.authenticateWithNodeRuntime(); ok {
		return res, tokens, err
	}

	nonce := generateNonce()

	if loginTicket, err := w.tryCrossOriginAuthenticate(); err == nil {
		authDebugf("cross-origin authenticate succeeded; login_ticket len=%d", len(loginTicket))
		code, err := w.authorizeWithLoginTicket(loginTicket, generateNonce(), nonce)
		if err == nil {
			tokens, err := w.exchangeCodeForTokens(code)
			if err == nil {
				res, err := w.loginToWeWork(tokens)
				return res, tokens, err
			}
		}
	}

	authParams := url.Values{}
	authParams.Add("redirect_uri", w.config.RedirectURI)
	authParams.Add("client_id", w.config.ClientID)
	authParams.Add("audience", w.config.Audience)
	authParams.Add("scope", "openid profile email offline_access")
	authParams.Add("response_type", "code")
	authParams.Add("response_mode", "query")
	authParams.Add("nonce", nonce)
	authParams.Add("code_challenge", w.codeChallenge)
	authParams.Add("code_challenge_method", "S256")
	authParams.Add("auth0Client", "eyJuYW1lIjoiQGF1dGgwL2F1dGgwLWFuZ3VsYXIiLCJ2ZXJzaW9uIjoiMS4xMS4xLmN1c3RvbSIsImVudiI6eyJhbmd1bGFyL2NvcmUiOiIxMy4xLjEifX0=")

	authURL := fmt.Sprintf("https://%s/authorize?%s", w.config.Domain, authParams.Encode())
	authDebugf("initial authorize GET %s", authURL)
	resp, err := w.client.Get(authURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get auth URL: %w", err)
	}
	authDebugf("initial authorize response status=%d request_url=%s location=%q", resp.StatusCode, requestURLString(resp), resp.Header.Get("Location"))

	if resp.Request != nil && resp.Request.URL != nil {
		if code, ok, err := extractCodeFromLocation(resp.Request.URL.String(), ""); err != nil {
			return nil, nil, err
		} else if ok {
			tokens, err := w.exchangeCodeForTokens(code)
			if err != nil {
				return nil, nil, err
			}
			res, err := w.loginToWeWork(tokens)
			return res, tokens, err
		}
	}

	loginLocation := resp.Header.Get("Location")
	if loginLocation == "" {
		if resp.Request != nil && resp.Request.URL != nil {
			loginLocation = resp.Request.URL.String()
		} else {
			return nil, nil, fmt.Errorf("no login URL in response")
		}
	}

	loginURL, err := resolveRelativeURL(resp.Request.URL, loginLocation)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve login URL: %w", err)
	}

	parsedLogin, err := url.Parse(loginURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse login URL: %w", err)
	}

	state := parsedLogin.Query().Get("state")

	code, err := w.followAuthorizationRedirects(resp, state)
	if err != nil {
		return nil, nil, err
	}

	tokens, err := w.exchangeCodeForTokens(code)
	if err != nil {
		return nil, nil, err
	}

	res, err := w.loginToWeWork(tokens)
	return res, tokens, err
}

func (w *WeWorkAuth) tryCrossOriginAuthenticate() (string, error) {
	bodyStruct := map[string]string{
		"client_id":       w.config.ClientID,
		"username":        w.username,
		"password":        w.password,
		"realm":           "id-wework",
		"credential_type": "http://auth0.com/oauth/grant-type/password-realm",
	}

	body, err := json.Marshal(bodyStruct)
	if err != nil {
		return "", fmt.Errorf("failed to marshal credential payload: %w", err)
	}

	url := fmt.Sprintf("https://%s/co/authenticate", w.config.Domain)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create credential request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", "https://members.wework.com")
	req.Header.Set("Referer", "https://members.wework.com/workplaceone/content2/login")
	req.Header.Set("Auth0-Client", "eyJuYW1lIjoiQGF1dGgwL2F1dGgwLWFuZ3VsYXIiLCJ2ZXJzaW9uIjoiMS4xMS4xLmN1c3RvbSIsImVudiI6eyJhbmd1bGFyL2NvcmUiOiIxMy4xLjEifX0=")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to authenticate credentials: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		LoginTicket      string `json:"login_ticket"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode credential response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if result.Error != "" {
			return "", fmt.Errorf("authentication failed: %s (%s)", result.ErrorDescription, result.Error)
		}
		return "", fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	if result.LoginTicket == "" {
		return "", fmt.Errorf("authentication failed: missing login ticket in response")
	}

	return result.LoginTicket, nil
}

func (w *WeWorkAuth) authorizeWithLoginTicket(loginTicket, state, nonce string) (string, error) {
	params := url.Values{}
	params.Add("redirect_uri", w.config.RedirectURI)
	params.Add("client_id", w.config.ClientID)
	params.Add("audience", w.config.Audience)
	params.Add("scope", "openid profile email offline_access")
	params.Add("response_type", "code")
	params.Add("response_mode", "query")
	params.Add("nonce", nonce)
	params.Add("state", state)
	params.Add("code_challenge", w.codeChallenge)
	params.Add("code_challenge_method", "S256")
	params.Add("auth0Client", "eyJuYW1lIjoiQGF1dGgwL2F1dGgwLWFuZ3VsYXIiLCJ2ZXJzaW9uIjoiMS4xMS4xLmN1c3RvbSIsImVudiI6eyJhbmd1bGFyL2NvcmUiOiIxMy4xLjEifX0=")
	params.Add("login_ticket", loginTicket)

	cookiePayload := map[string]string{
		"nonce":         nonce,
		"code_verifier": w.codeVerifier,
		"scope":         "openid profile email offline_access",
		"audience":      w.config.Audience,
		"redirect_uri":  w.config.RedirectURI,
		"state":         state,
	}

	if payloadBytes, err := json.Marshal(cookiePayload); err == nil {
		cookieValue := url.QueryEscape(string(payloadBytes))
		domainURL, err := url.Parse(fmt.Sprintf("https://%s", w.config.Domain))
		if err == nil {
			cookies := []*http.Cookie{
				{
					Name:  fmt.Sprintf("_legacy_a0.spajs.txs.%s", w.config.ClientID),
					Value: cookieValue,
					Path:  "/",
				},
				{
					Name:  fmt.Sprintf("a0.spajs.txs.%s", w.config.ClientID),
					Value: cookieValue,
					Path:  "/",
				},
			}
			w.client.Jar.SetCookies(domainURL, cookies)
		}
	}

	authURL := fmt.Sprintf("https://%s/authorize?%s", w.config.Domain, params.Encode())
	authDebugf("authorizeWithLoginTicket GET %s", authURL)
	resp, err := w.client.Get(authURL)
	if err != nil {
		return "", fmt.Errorf("failed to initiate authorization: %w", err)
	}
	defer resp.Body.Close()
	authDebugf("authorizeWithLoginTicket response status=%d request_url=%s location=%q", resp.StatusCode, requestURLString(resp), resp.Header.Get("Location"))

	if resp.Request != nil && resp.Request.URL != nil {
		if code, ok, err := extractCodeFromLocation(resp.Request.URL.String(), state); err != nil {
			return "", err
		} else if ok {
			return code, nil
		}
	}

	return w.followAuthorizationRedirects(resp, state)
}

func (w *WeWorkAuth) fetchForm(pageURL string) (string, string, url.Values, string, error) {
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return "", "", nil, "", fmt.Errorf("failed to create form request: %w", err)
	}

	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	resp, err := w.client.Do(req)
	if err != nil {
		return "", "", nil, "", fmt.Errorf("failed to load form page %s: %w", pageURL, err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", "", nil, "", fmt.Errorf("failed to parse form page: %w", err)
	}

	form := doc.Find("form").First()
	if form.Length() == 0 {
		return "", "", nil, "", fmt.Errorf("no form found at %s", pageURL)
	}

	action, method, values, err := extractForm(form, resp.Request.URL)
	if err != nil {
		return "", "", nil, "", err
	}

	return action, method, values, resp.Request.URL.String(), nil
}

func (w *WeWorkAuth) submitForm(method, actionURL, referer string, values url.Values) (*http.Response, error) {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodPost
	}

	var body io.Reader
	requestURL := actionURL
	if method == http.MethodGet {
		sep := "?"
		if strings.Contains(requestURL, "?") {
			sep = "&"
		}
		requestURL += sep + values.Encode()
	} else {
		body = strings.NewReader(values.Encode())
	}

	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create form submission request: %w", err)
	}

	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	if parsed, err := url.Parse(actionURL); err == nil {
		req.Header.Set("Origin", fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host))
	}

	return w.client.Do(req)
}

func applyLoginFormDefaults(values url.Values) {
	if values.Get("js-available") != "" {
		values.Set("js-available", "true")
	}
	if values.Get("webauthn-available") != "" {
		values.Set("webauthn-available", "false")
	}
	if values.Get("webauthn-platform-available") != "" {
		values.Set("webauthn-platform-available", "false")
	}
	if values.Get("is-brave") != "" {
		values.Set("is-brave", "false")
	}
}

func resolveRelativeURL(base *url.URL, ref string) (string, error) {
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return ref, nil
	}

	if base == nil {
		return "", fmt.Errorf("cannot resolve relative URL %s without base", ref)
	}

	resolved, err := base.Parse(ref)
	if err != nil {
		return "", err
	}

	return resolved.String(), nil
}

func extractForm(form *goquery.Selection, base *url.URL) (string, string, url.Values, error) {
	action, exists := form.Attr("action")
	if !exists || action == "" {
		action = base.String()
	} else {
		absoluteAction, err := resolveRelativeURL(base, action)
		if err != nil {
			return "", "", nil, fmt.Errorf("failed to resolve form action: %w", err)
		}
		action = absoluteAction
	}

	method, _ := form.Attr("method")
	values := url.Values{}
	form.Find("input").Each(func(_ int, s *goquery.Selection) {
		name, exists := s.Attr("name")
		if !exists || name == "" {
			return
		}

		if value, ok := s.Attr("value"); ok {
			values.Set(name, value)
		} else {
			values.Add(name, "")
		}
	})

	return action, method, values, nil
}

func (w *WeWorkAuth) followAuthorizationRedirects(initialResp *http.Response, expectedState string) (string, error) {
	currentResp := initialResp
	retryCount := 0

	for {
		authDebugf("followAuthorizationRedirects status=%d request_url=%s location=%q", currentResp.StatusCode, requestURLString(currentResp), currentResp.Header.Get("Location"))
		if currentResp.StatusCode >= 300 && currentResp.StatusCode < 400 {
			location := currentResp.Header.Get("Location")
			if location == "" {
				body, _ := io.ReadAll(currentResp.Body)
				currentResp.Body.Close()
				return "", fmt.Errorf("authorization redirect missing location: status %d body %s", currentResp.StatusCode, clipBody(body))
			}

			if code, ok, err := extractCodeFromLocation(location, expectedState); err != nil {
				currentResp.Body.Close()
				return "", err
			} else if ok {
				currentResp.Body.Close()
				return code, nil
			}

			nextURL := location
			if !strings.HasPrefix(nextURL, "http") {
				nextURL = fmt.Sprintf("https://%s%s", w.config.Domain, nextURL)
			}

			nextResp, err := w.client.Get(nextURL)
			if err != nil {
				currentResp.Body.Close()
				return "", fmt.Errorf("failed to follow redirect: %w", err)
			}
			currentResp.Body.Close()
			currentResp = nextResp
			continue
		}

		if currentResp.StatusCode == http.StatusTooManyRequests {
			retryAfter := time.Second * 2
			if ra := currentResp.Header.Get("Retry-After"); ra != "" {
				if seconds, err := time.ParseDuration(ra + "s"); err == nil {
					retryAfter = seconds
				}
			}

			if retryCount >= 3 {
				body, _ := io.ReadAll(currentResp.Body)
				currentResp.Body.Close()
				return "", fmt.Errorf("authorization rate limited after retries: %s", clipBody(body))
			}

			retryCount++
			_, _ = io.ReadAll(currentResp.Body)
			currentResp.Body.Close()
			time.Sleep(retryAfter)

			retryURL := currentResp.Request.URL
			if retryURL == nil {
				return "", fmt.Errorf("rate limited without request URL")
			}

			nextResp, err := w.client.Get(retryURL.String())
			if err != nil {
				return "", fmt.Errorf("failed to retry after rate limit: %w", err)
			}
			currentResp = nextResp
			continue
		}

		if currentResp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(currentResp.Body)
			if err != nil {
				currentResp.Body.Close()
				return "", fmt.Errorf("failed to read intermediate page: %w", err)
			}
			currentResp.Body.Close()

			nextResp, handled, err := w.handleIntermediatePage(body, currentResp.Request.URL)
			if err != nil {
				return "", err
			}
			if handled {
				currentResp = nextResp
				continue
			}

			return "", fmt.Errorf("authorization did not return a code: status %d body %s", http.StatusOK, clipBody(body))
		}

		body, _ := io.ReadAll(currentResp.Body)
		currentResp.Body.Close()
		return "", fmt.Errorf("authorization did not return a code: status %d body %s", currentResp.StatusCode, clipBody(body))
	}
}

func (w *WeWorkAuth) handleIntermediatePage(body []byte, baseURL *url.URL) (*http.Response, bool, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil, false, nil
	}

	type formCandidate struct {
		selection *goquery.Selection
		kind      string
	}

	candidates := make([]formCandidate, 0)
	doc.Find("form").Each(func(_ int, s *goquery.Selection) {
		switch {
		case s.Find("input[name='password']").Length() > 0:
			candidates = append(candidates, formCandidate{selection: s, kind: "password"})
		case s.Find("input[name='js-available']").Length() > 0:
			candidates = append(candidates, formCandidate{selection: s, kind: "detection"})
		case s.Find("input[name='username']").Length() > 0:
			candidates = append(candidates, formCandidate{selection: s, kind: "identifier"})
		}
	})

	if len(candidates) == 0 {
		return nil, false, nil
	}

	var selected formCandidate
	for _, c := range candidates {
		if c.kind == "password" {
			selected = c
			break
		}
		if c.kind == "detection" && selected.selection == nil {
			selected = c
		}
		if c.kind == "identifier" && selected.selection == nil {
			selected = c
		}
	}

	action, method, values, err := extractForm(selected.selection, baseURL)
	if err != nil {
		return nil, false, err
	}
	authDebugf("handleIntermediatePage kind=%s method=%s base=%s action=%s has_username=%t has_password=%t has_state=%t", selected.kind, method, func() string {
		if baseURL == nil {
			return ""
		}
		return baseURL.String()
	}(), action, values.Get("username") != "", values.Get("password") != "", values.Get("state") != "")

	switch selected.kind {
	case "identifier":
		values.Set("username", w.username)
	case "password":
		values.Set("password", w.password)
	case "detection":
		applyLoginFormDefaults(values)
		if values.Get("action") == "" {
			values.Set("action", "default")
		}
	}

	referer := ""
	if baseURL != nil {
		referer = baseURL.String()
	}

	resp, err := w.submitForm(method, action, referer, values)
	if err != nil {
		return nil, false, err
	}
	authDebugf("handleIntermediatePage submit response status=%d request_url=%s location=%q", resp.StatusCode, requestURLString(resp), resp.Header.Get("Location"))

	return resp, true, nil
}

func extractCodeFromLocation(location, expectedState string) (string, bool, error) {
	parsed, err := url.Parse(location)
	if err != nil {
		return "", false, fmt.Errorf("failed to parse redirect URL: %w", err)
	}

	query := parsed.Query()
	code := query.Get("code")
	if code == "" {
		return "", false, nil
	}

	if expectedState != "" && query.Get("state") != expectedState {
		return "", false, fmt.Errorf("state mismatch in authorization response")
	}

	return code, true, nil
}

func (w *WeWorkAuth) exchangeCodeForTokens(code string) (*OAuthTokenResponse, error) {
	tokenData := map[string]string{
		"client_id":     w.config.ClientID,
		"code_verifier": w.codeVerifier,
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  w.config.RedirectURI,
	}

	body, err := json.Marshal(tokenData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal token payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://%s/oauth/token", w.config.Domain), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}
	defer resp.Body.Close()

	var tokens OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	return &tokens, nil
}

func requestURLString(resp *http.Response) string {
	if resp == nil || resp.Request == nil || resp.Request.URL == nil {
		return ""
	}
	return resp.Request.URL.String()
}

func clipBody(body []byte) string {
	const limit = 512
	if len(body) > limit {
		return string(body[:limit]) + "..."
	}
	return string(body)
}

func (w *WeWorkAuth) loginToWeWork(tokens *OAuthTokenResponse) (*LoginByAuth0TokenResponse, error) {
	loginURL := "https://members.wework.com/workplaceone/api/auth0/login-by-auth0-token"
	loginData := map[string]any{
		"id_token":      tokens.IDToken,
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"expires_in":    tokens.ExpiresIn,
		"scope":         tokens.Scope,
		"token_type":    tokens.TokenType,
		"client_id":     w.config.ClientID,
		"audience":      w.config.Audience,
	}

	loginBody, err := json.Marshal(loginData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal WeWork login data: %v", err)
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(loginBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create WeWork login request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Request-Source", "com.wework.ondemand/WorkplaceOne/Prod/iOS/2.71.0(26.1)")
	req.Header.Set("User-Agent", "Mobile Safari 16.1")

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to login to WeWork: %v", err)
	}
	defer resp.Body.Close()

	var loginResp LoginByAuth0TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("failed to decode WeWork login response: %v", err)
	}

	return &loginResp, nil
}

func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func generateNonce() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}
