package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	webvpnBaseURL           = "https://webvpn.zju.edu.cn"
	loginURL                = webvpnBaseURL + "/login"
	doLoginURL              = webvpnBaseURL + "/do-login"
	doConfirmLoginURL       = webvpnBaseURL + "/do-confirm-login"
	defaultRouteValue       = "8768cab8c7e7ee1c6799ad807f94da0a"
	defaultCookieCacheFile  = ".webvpn-cookie-cache.json"
	defaultTokenPath        = "/https/77726476706e69737468656265737421ffe744922e3426537d51d1e2974724/connect/token?vpn-12-o2-www.cc98.org"
	defaultSignPath         = "/https/77726476706e69737468656265737421f1e748d22433310830079bab/me/signin?vpn-12-o2-www.cc98.org"
)

var (
	csrfPattern          = regexp.MustCompile(`name="_csrf"\s+value="([^"]+)"`)
	captchaIDPattern     = regexp.MustCompile(`name="captcha_id"\s+value="([^"]+)"`)
	passwordKeyIVPattern = regexp.MustCompile(`encrypt\s*\([^,]+,\s*"([^"]+)"\s*,\s*"([^"]+)"\s*\)`)
)

type loginPageFields struct {
	CSRF        string
	CaptchaID   string
	PasswordKey string
	PasswordIV  string
}

type cookieCache struct {
	SavedAt int64             `json:"saved_at"`
	Cookies map[string]string `json:"cookies"`
}

func newCookieJar() http.CookieJar {
	jar, _ := cookiejar.New(nil)
	return jar
}

func cookieCachePath(envPath string) string {
	return filepath.Join(filepath.Dir(envPath), defaultCookieCacheFile)
}

func webvpnURL() *url.URL {
	u, _ := url.Parse(webvpnBaseURL)
	return u
}

func exportCookieMap(jar http.CookieJar) map[string]string {
	values := map[string]string{}
	for _, cookie := range jar.Cookies(webvpnURL()) {
		values[cookie.Name] = cookie.Value
	}
	return values
}

func ensureDefaultRouteCookie(jar http.CookieJar) {
	values := exportCookieMap(jar)
	if strings.TrimSpace(values["route"]) != "" {
		return
	}
	jar.SetCookies(webvpnURL(), []*http.Cookie{{
		Name:   "route",
		Value:  defaultRouteValue,
		Domain: "webvpn.zju.edu.cn",
		Path:   "/",
	}})
}

func restoreCookieMap(jar http.CookieJar, values map[string]string) {
	cookies := make([]*http.Cookie, 0, len(values)+1)
	for name, value := range values {
		if strings.TrimSpace(name) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		cookies = append(cookies, &http.Cookie{
			Name:   name,
			Value:  value,
			Domain: "webvpn.zju.edu.cn",
			Path:   "/",
		})
	}
	if len(cookies) > 0 {
		jar.SetCookies(webvpnURL(), cookies)
	}
	ensureDefaultRouteCookie(jar)
}

func hasCoreWebVPNCookies(jar http.CookieJar) bool {
	values := exportCookieMap(jar)
	return strings.TrimSpace(values["wengine_vpn_ticketwebvpn_zju_edu_cn"]) != ""
}

func loadCookieCache(jar http.CookieJar, path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	var payload cookieCache
	if err := json.Unmarshal(content, &payload); err != nil || len(payload.Cookies) == 0 {
		return false
	}

	restoreCookieMap(jar, payload.Cookies)
	return hasCoreWebVPNCookies(jar)
}

func saveCookieCache(jar http.CookieJar, path string) error {
	payload := cookieCache{
		SavedAt: unixNow(),
		Cookies: exportCookieMap(jar),
	}
	content, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o600)
}

func clearWebVPNCookies(jar http.CookieJar) {
	current := exportCookieMap(jar)
	expired := make([]*http.Cookie, 0, len(current))
	for name := range current {
		expired = append(expired, &http.Cookie{
			Name:   name,
			Value:  "",
			Path:   "/",
			Domain: "webvpn.zju.edu.cn",
			MaxAge: -1,
		})
	}
	if len(expired) > 0 {
		jar.SetCookies(webvpnURL(), expired)
	}
}

func loginWebVPN(ctx context.Context, client *http.Client, username, password string, recordTiming func(string, time.Duration)) error {
	loginStarted := time.Now()
	loginPage, err := fetchLoginPage(ctx, client)
	if recordTiming != nil {
		recordTiming("GET /login", time.Since(loginStarted))
	}
	if err != nil {
		return err
	}

	fields, err := parseLoginPage(loginPage)
	if err != nil {
		return err
	}

	encryptedPassword, err := encryptPassword(password, fields.PasswordKey, fields.PasswordIV)
	if err != nil {
		return err
	}

	form := url.Values{}
	form.Set("_csrf", fields.CSRF)
	form.Set("auth_type", "local")
	form.Set("username", username)
	form.Set("sms_code", "")
	form.Set("password", encryptedPassword)
	form.Set("captcha", "")
	form.Set("needCaptcha", "false")
	form.Set("captcha_id", fields.CaptchaID)

	doLoginStarted := time.Now()
	body, statusCode, err := postForm(ctx, client, doLoginURL, form)
	if recordTiming != nil {
		recordTiming("POST /do-login", time.Since(doLoginStarted))
	}
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("unexpected do-login status %d", statusCode)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		payload = map[string]any{}
	}

	loginError := stringValue(payload["error"])
	switch {
	case loginError == "NEED_CONFIRM":
		confirmStarted := time.Now()
		_, statusCode, err = postForm(ctx, client, doConfirmLoginURL, form)
		if recordTiming != nil {
			recordTiming("POST /do-confirm-login", time.Since(confirmStarted))
		}
		if err != nil {
			return err
		}
		if statusCode != http.StatusOK {
			return fmt.Errorf("unexpected do-confirm-login status %d", statusCode)
		}
	case loginError != "":
		return fmt.Errorf("webvpn login failed: %s", firstNonEmpty(stringValue(payload["message"]), loginError))
	}

	ensureDefaultRouteCookie(client.Jar)
	if !hasCoreWebVPNCookies(client.Jar) {
		return errors.New("missing wengine_vpn_ticketwebvpn_zju_edu_cn after login")
	}
	return nil
}

func fetchLoginPage(ctx context.Context, client *http.Client) (string, error) {
	body, statusCode, err := get(ctx, client, loginURL, nil)
	if err != nil {
		return "", err
	}
	if statusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected login page status %d", statusCode)
	}
	return string(body), nil
}

func parseLoginPage(body string) (loginPageFields, error) {
	csrfMatch := csrfPattern.FindStringSubmatch(body)
	captchaMatch := captchaIDPattern.FindStringSubmatch(body)
	passwordMatch := passwordKeyIVPattern.FindStringSubmatch(body)
	if len(csrfMatch) != 2 || len(captchaMatch) != 2 || len(passwordMatch) != 3 {
		return loginPageFields{}, errors.New("missing webvpn login page fields")
	}
	return loginPageFields{
		CSRF:        csrfMatch[1],
		CaptchaID:   captchaMatch[1],
		PasswordKey: passwordMatch[1],
		PasswordIV:  passwordMatch[2],
	}, nil
}

func encryptPassword(password, key, iv string) (string, error) {
	keyBytes := []byte(key)
	ivBytes := []byte(iv)
	if len(keyBytes) != 16 || len(ivBytes) != aes.BlockSize {
		return "", errors.New("invalid AES key or IV length")
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}

	plainBytes := []byte(password)
	encrypted := make([]byte, len(plainBytes))
	stream := cipher.NewCFBEncrypter(block, ivBytes)
	stream.XORKeyStream(encrypted, plainBytes)

	return hex.EncodeToString(ivBytes) + hex.EncodeToString(encrypted), nil
}

func postForm(ctx context.Context, client *http.Client, targetURL string, form url.Values) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}
