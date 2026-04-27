//go:build !js

package wework

func (w *WeWorkAuth) authenticateWithNodeRuntime() (*LoginByAuth0TokenResponse, *OAuthTokenResponse, bool, error) {
	return nil, nil, false, nil
}
