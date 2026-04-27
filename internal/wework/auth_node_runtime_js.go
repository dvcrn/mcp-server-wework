//go:build js

package wework

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gopherjs/gopherjs/js"
)

type nodeCookieJar struct {
	values map[string]string
}

func newNodeCookieJar() *nodeCookieJar {
	return &nodeCookieJar{values: map[string]string{}}
}

func (j *nodeCookieJar) Set(name, value string) {
	if strings.TrimSpace(name) == "" {
		return
	}
	j.values[name] = value
}

func (j *nodeCookieJar) AddSetCookie(header string) {
	parts := strings.SplitN(header, ";", 2)
	if len(parts) == 0 {
		return
	}
	pair := strings.TrimSpace(parts[0])
	idx := strings.Index(pair, "=")
	if idx <= 0 {
		return
	}
	j.Set(strings.TrimSpace(pair[:idx]), pair[idx+1:])
}

func (j *nodeCookieJar) Header() string {
	keys := make([]string, 0, len(j.values))
	for key := range j.values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+j.values[key])
	}
	return strings.Join(parts, "; ")
}

type nodeResponse struct {
	Status     int
	URL        string
	Location   string
	Body       string
	SetCookies []string
}

func (w *WeWorkAuth) authenticateWithNodeRuntime() (*LoginByAuth0TokenResponse, *OAuthTokenResponse, bool, error) {
	if js.Global.Get("process") == js.Undefined || js.Global.Get("fetch") == js.Undefined {
		return nil, nil, false, nil
	}

	jar := newNodeCookieJar()
	jar.Set("auth0.zE51Ep7FttlmtQV6ZEGyJKsY2jD1EtAu.is.authenticated", "true")
	jar.Set("_legacy_auth0.zE51Ep7FttlmtQV6ZEGyJKsY2jD1EtAu.is.authenticated", "true")

	loginTicket, err := w.nodeTryCrossOriginAuthenticate(jar)
	if err != nil {
		return nil, nil, true, err
	}
	authDebugf("node runtime auth succeeded; login_ticket len=%d", len(loginTicket))

	state := generateNonce()
	nonce := generateNonce()
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
		jar.Set("_legacy_a0.spajs.txs."+w.config.ClientID, cookieValue)
		jar.Set("a0.spajs.txs."+w.config.ClientID, cookieValue)
	}

	code, err := w.nodeAuthorize(jar, loginTicket, state, nonce)
	if err != nil {
		return nil, nil, true, err
	}

	tokens, err := w.nodeExchangeCodeForTokens(jar, code)
	if err != nil {
		return nil, nil, true, err
	}
	login, err := w.nodeLoginToWeWork(jar, tokens)
	if err != nil {
		return nil, nil, true, err
	}
	return login, tokens, true, nil
}

func (w *WeWorkAuth) nodeTryCrossOriginAuthenticate(jar *nodeCookieJar) (string, error) {
	bodyStruct := map[string]string{
		"client_id":       w.config.ClientID,
		"username":        w.username,
		"password":        w.password,
		"realm":           "id-wework",
		"credential_type": "http://auth0.com/oauth/grant-type/password-realm",
	}
	body, err := json.Marshal(bodyStruct)
	if err != nil {
		return "", err
	}

	resp, err := nodeFetch("POST", fmt.Sprintf("https://%s/co/authenticate", w.config.Domain), map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
		"Origin":       "https://members.wework.com",
		"Referer":      "https://members.wework.com/workplaceone/content2/login",
		"Auth0-Client": "eyJuYW1lIjoiQGF1dGgwL2F1dGgwLWFuZ3VsYXIiLCJ2ZXJzaW9uIjoiMS4xMS4xLmN1c3RvbSIsImVudiI6eyJhbmd1bGFyL2NvcmUiOiIxMy4xLjEifX0=",
	}, string(body), jar, true)
	if err != nil {
		return "", err
	}
	for _, cookie := range resp.SetCookies {
		jar.AddSetCookie(cookie)
	}

	var result struct {
		LoginTicket      string `json:"login_ticket"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal([]byte(resp.Body), &result); err != nil {
		return "", fmt.Errorf("failed to decode credential response: %w", err)
	}
	if resp.Status != http.StatusOK {
		if result.Error != "" {
			return "", fmt.Errorf("authentication failed: %s (%s)", result.ErrorDescription, result.Error)
		}
		return "", fmt.Errorf("authentication failed with status %d", resp.Status)
	}
	if result.LoginTicket == "" {
		return "", fmt.Errorf("authentication failed: missing login ticket in response")
	}
	return result.LoginTicket, nil
}

func (w *WeWorkAuth) nodeAuthorize(jar *nodeCookieJar, loginTicket, state, nonce string) (string, error) {
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
	if loginTicket != "" {
		params.Add("login_ticket", loginTicket)
	}

	currentURL := fmt.Sprintf("https://%s/authorize?%s", w.config.Domain, params.Encode())
	resp, err := nodeFetch("GET", currentURL, map[string]string{}, "", jar, true)
	if err != nil {
		return "", err
	}
	for _, cookie := range resp.SetCookies {
		jar.AddSetCookie(cookie)
	}

	for {
		authDebugf("nodeAuthorize status=%d url=%s location=%q", resp.Status, currentURL, resp.Location)
		if code, ok, err := extractCodeFromLocation(nodeFirstNonEmpty(resp.Location, resp.URL, currentURL), state); err != nil {
			return "", err
		} else if ok {
			return code, nil
		}

		switch {
		case resp.Status >= 300 && resp.Status < 400:
			if resp.Location == "" {
				return "", fmt.Errorf("authorization redirect missing location: status %d body %s", resp.Status, clipBody([]byte(resp.Body)))
			}
			nextURL, err := resolveRelativeURL(mustParseURL(currentURL), resp.Location)
			if err != nil {
				return "", err
			}
			currentURL = nextURL
			resp, err = nodeFetch("GET", currentURL, map[string]string{}, "", jar, true)
			if err != nil {
				return "", err
			}
			for _, cookie := range resp.SetCookies {
				jar.AddSetCookie(cookie)
			}
		case resp.Status == http.StatusOK:
			previousURL := currentURL
			nextMethod, nextURL, nextBody, handled, err := nodeHandleIntermediatePage(resp.Body, currentURL, w.username, w.password)
			if err != nil {
				return "", err
			}
			if !handled {
				return "", fmt.Errorf("authorization did not return a code: status %d body %s", resp.Status, clipBody([]byte(resp.Body)))
			}
			currentURL = nextURL
			headers := map[string]string{
				"Accept":         "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
				"Referer":        previousURL,
				"Sec-Fetch-Mode": "navigate",
				"Sec-Fetch-Dest": "document",
				"Sec-Fetch-Site": "same-origin",
			}
			if nextBody != "" && nextMethod != http.MethodGet {
				headers["Content-Type"] = "application/x-www-form-urlencoded"
			}
			if parsed, parseErr := url.Parse(nextURL); parseErr == nil {
				headers["Origin"] = fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
			}
			resp, err = nodeFetch(nextMethod, nextURL, headers, nextBody, jar, true)
			if err != nil {
				return "", err
			}
			for _, cookie := range resp.SetCookies {
				jar.AddSetCookie(cookie)
			}
		default:
			return "", fmt.Errorf("authorization did not return a code: status %d body %s", resp.Status, clipBody([]byte(resp.Body)))
		}
	}
}

func nodeHandleIntermediatePage(body, baseURL, username, password string) (method, action, encodedBody string, handled bool, err error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(body))
	if err != nil {
		return "", "", "", false, nil
	}

	type formCandidate struct {
		selection *goquery.Selection
		kind      string
	}
	var candidates []formCandidate
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
		return "", "", "", false, nil
	}

	selected := candidates[0]
	for _, c := range candidates {
		if c.kind == "password" {
			selected = c
			break
		}
		if c.kind == "detection" && selected.kind != "password" {
			selected = c
		}
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return "", "", "", false, err
	}
	action, formMethod, values, err := extractForm(selected.selection, parsedBase)
	if err != nil {
		return "", "", "", false, err
	}
	switch selected.kind {
	case "identifier":
		values.Set("username", username)
	case "password":
		values.Set("password", password)
	case "detection":
		applyLoginFormDefaults(values)
		if values.Get("action") == "" {
			values.Set("action", "default")
		}
	}
	authDebugf("nodeHandleIntermediatePage kind=%s method=%s action=%s encoded=%s", selected.kind, formMethod, action, values.Encode())
	if strings.TrimSpace(formMethod) == "" {
		formMethod = http.MethodPost
	}
	if strings.EqualFold(formMethod, http.MethodGet) {
		sep := "?"
		if strings.Contains(action, "?") {
			sep = "&"
		}
		action += sep + values.Encode()
		return http.MethodGet, action, "", true, nil
	}
	return strings.ToUpper(formMethod), action, values.Encode(), true, nil
}

func (w *WeWorkAuth) nodeExchangeCodeForTokens(jar *nodeCookieJar, code string) (*OAuthTokenResponse, error) {
	bodyMap := map[string]string{
		"client_id":     w.config.ClientID,
		"code_verifier": w.codeVerifier,
		"grant_type":    "authorization_code",
		"code":          code,
		"redirect_uri":  w.config.RedirectURI,
	}
	body, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}
	resp, err := nodeFetch("POST", fmt.Sprintf("https://%s/oauth/token", w.config.Domain), map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json",
	}, string(body), jar, true)
	if err != nil {
		return nil, err
	}
	var tokens OAuthTokenResponse
	if err := json.Unmarshal([]byte(resp.Body), &tokens); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}
	if resp.Status != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.Status)
	}
	return &tokens, nil
}

func (w *WeWorkAuth) nodeLoginToWeWork(jar *nodeCookieJar, tokens *OAuthTokenResponse) (*LoginByAuth0TokenResponse, error) {
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
	body, err := json.Marshal(loginData)
	if err != nil {
		return nil, err
	}
	resp, err := nodeFetch("POST", "https://members.wework.com/workplaceone/api/auth0/login-by-auth0-token", map[string]string{
		"Content-Type":   "application/json",
		"Request-Source": "com.wework.ondemand/WorkplaceOne/Prod/iOS/2.71.0(26.1)",
		"User-Agent":     "Mobile Safari 16.1",
	}, string(body), jar, true)
	if err != nil {
		return nil, err
	}
	var loginResp LoginByAuth0TokenResponse
	if err := json.Unmarshal([]byte(resp.Body), &loginResp); err != nil {
		return nil, fmt.Errorf("failed to decode WeWork login response: %w", err)
	}
	return &loginResp, nil
}

func nodeFetch(method, requestURL string, headers map[string]string, body string, jar *nodeCookieJar, manualRedirect bool) (*nodeResponse, error) {
	headersObj := js.Global.Get("Object").New()
	for key, value := range headers {
		headersObj.Set(key, value)
	}
	if jar != nil {
		if cookieHeader := jar.Header(); cookieHeader != "" {
			headersObj.Set("Cookie", cookieHeader)
		}
	}

	opts := js.Global.Get("Object").New()
	opts.Set("method", method)
	opts.Set("headers", headersObj)
	if manualRedirect {
		opts.Set("redirect", "manual")
	}
	if body != "" {
		opts.Set("body", body)
	}

	respObj, err := awaitObjectPromise(js.Global.Call("fetch", requestURL, opts))
	if err != nil {
		return nil, err
	}
	text, err := awaitStringPromise(respObj.Call("text"))
	if err != nil {
		return nil, err
	}

	resp := &nodeResponse{
		Status:   respObj.Get("status").Int(),
		URL:      respObj.Get("url").String(),
		Location: normalizeNodeHeaderValue(respObj.Get("headers").Call("get", "location").String()),
		Body:     text,
	}

	headersJS := respObj.Get("headers")
	getSetCookie := headersJS.Get("getSetCookie")
	if getSetCookie != js.Undefined && getSetCookie != nil {
		cookies := headersJS.Call("getSetCookie")
		for i := 0; i < cookies.Length(); i++ {
			resp.SetCookies = append(resp.SetCookies, cookies.Index(i).String())
		}
	}
	return resp, nil
}

func awaitObjectPromise(promise *js.Object) (*js.Object, error) {
	type result struct {
		value *js.Object
		err   error
	}
	ch := make(chan result, 1)
	promise.Call("then", func(value *js.Object) {
		go func() { ch <- result{value: value} }()
	}).Call("catch", func(err *js.Object) {
		go func() { ch <- result{err: fmt.Errorf(jsErrorString(err))} }()
	})
	res := <-ch
	return res.value, res.err
}

func awaitStringPromise(promise *js.Object) (string, error) {
	type result struct {
		value string
		err   error
	}
	ch := make(chan result, 1)
	promise.Call("then", func(value string) {
		go func() { ch <- result{value: value} }()
	}).Call("catch", func(err *js.Object) {
		go func() { ch <- result{err: fmt.Errorf(jsErrorString(err))} }()
	})
	res := <-ch
	return res.value, res.err
}

func normalizeNodeHeaderValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "null" || value == "undefined" {
		return ""
	}
	return value
}

func jsErrorString(err *js.Object) string {
	if err == nil || err == js.Undefined {
		return "javascript promise rejected"
	}
	if message := err.Get("message"); message != js.Undefined && message != nil && message.String() != "" {
		return message.String()
	}
	return err.String()
}

func nodeFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func mustParseURL(raw string) *url.URL {
	parsed, _ := url.Parse(raw)
	return parsed
}
