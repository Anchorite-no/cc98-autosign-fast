package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	cc98ClientID           = "9a1fd200-8687-44b1-4c20-08d50a96e5cd"
	cc98ClientSecret       = "8b53f727-08e2-4509-8857-e34bf92b27f2"
	defaultUserAgent       = "cc98-signin/1.0"
	defaultConnectTimeout  = 5 * time.Second
	defaultRequestTimeout  = 15 * time.Second
)

type Runner struct {
	cfg       Config
	client    *http.Client
	output    io.Writer
	cachePath string
}

type accountResult struct {
	Index       int
	ResultText  string
	TokenResult tokenResult
}

type tokenResult struct {
	AccessToken string
	RawText     string
}

type signResult struct {
	StatusCode int
	Body       string
	Payload    map[string]any
	OK         bool
}

type signSummary struct {
	Status string
	Reward *int
	Streak *int
}

func NewRunner(cfg Config, output io.Writer) *Runner {
	return &Runner{
		cfg:       cfg,
		client:    newHTTPClient(),
		output:    output,
		cachePath: cookieCachePath(cfg.EnvPath),
	}
}

func (r *Runner) Run(ctx context.Context) error {
	started := time.Now()
	cookieCacheLoaded := loadCookieCache(r.client.Jar, r.cachePath)
	cookieCacheHit := cookieCacheLoaded

	if !cookieCacheLoaded {
		if err := loginWebVPN(ctx, r.client, r.cfg.WebVPNUser, r.cfg.WebVPNPass); err != nil {
			return err
		}
		if err := saveCookieCache(r.client.Jar, r.cachePath); err != nil {
			return err
		}
	}

	results := make([]accountResult, 0, len(r.cfg.Accounts))
	for _, account := range r.cfg.Accounts {
		result, retried, err := r.runAccountWithCookieRetry(ctx, account, cookieCacheLoaded && account.Index == 1)
		if err != nil {
			return err
		}
		if retried {
			cookieCacheHit = false
		}
		results = append(results, result)
	}

	for _, line := range formatOutputLines(results) {
		fmt.Fprintln(r.output, line)
	}
	if cookieCacheHit {
		fmt.Fprintln(r.output, "Cookie ✅ 命中")
	} else {
		fmt.Fprintln(r.output, "Cookie ❌ 未命中")
	}
	fmt.Fprintf(r.output, "耗时 ⏱ %.2fs\n", time.Since(started).Seconds())
	return nil
}

func (r *Runner) runAccountWithCookieRetry(ctx context.Context, account Account, allowCookieRetry bool) (accountResult, bool, error) {
	result := r.runAccount(ctx, account)
	if !allowCookieRetry || !isWebVPNLoginResponse(result.TokenResult) {
		return result, false, nil
	}

	clearWebVPNCookies(r.client.Jar)
	if err := loginWebVPN(ctx, r.client, r.cfg.WebVPNUser, r.cfg.WebVPNPass); err != nil {
		return accountResult{}, false, err
	}
	if err := saveCookieCache(r.client.Jar, r.cachePath); err != nil {
		return accountResult{}, false, err
	}
	return r.runAccount(ctx, account), true, nil
}

func (r *Runner) runAccount(ctx context.Context, account Account) accountResult {
	token := r.fetchAccessToken(ctx, account)
	if token.AccessToken == "" {
		return accountResult{
			Index:       account.Index,
			ResultText:  formatFailureText("获取 token 失败", token.RawText),
			TokenResult: token,
		}
	}

	sign := r.signIn(ctx, token.AccessToken)
	var info map[string]any
	if sign.OK {
		info = r.getSignInfo(ctx, token.AccessToken)
	}

	return accountResult{
		Index:       account.Index,
		ResultText:  formatResultText(summarizeSignResult(sign, info), sign),
		TokenResult: token,
	}
}

func (r *Runner) fetchAccessToken(ctx context.Context, account Account) tokenResult {
	values := urlValues(
		"client_id", cc98ClientID,
		"client_secret", cc98ClientSecret,
		"grant_type", "password",
		"username", account.Username,
		"password", account.Password,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webvpnBaseURL+defaultTokenPath, strings.NewReader(values))
	if err != nil {
		return tokenResult{RawText: "token_request_build_failed"}
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://www.cc98.org")
	req.Header.Set("Referer", "https://www.cc98.org/")
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		return tokenResult{RawText: classifyTransportError("token", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tokenResult{RawText: "token_read_failed"}
	}

	rawText := strings.TrimSpace(string(body))
	if resp.StatusCode != http.StatusOK {
		return tokenResult{RawText: firstNonEmpty(parseErrorMessage(body), rawText, fmt.Sprintf("HTTP %d", resp.StatusCode))}
	}

	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return tokenResult{RawText: rawText}
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return tokenResult{RawText: rawText}
	}

	return tokenResult{
		AccessToken: payload.AccessToken,
		RawText:     rawText,
	}
}

func (r *Runner) signIn(ctx context.Context, accessToken string) signResult {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webvpnBaseURL+defaultSignPath, bytes.NewReader(nil))
	if err != nil {
		return signResult{StatusCode: 0, Body: "signin_request_build_failed"}
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Origin", "https://www.cc98.org")
	req.Header.Set("Referer", "https://www.cc98.org/")
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := r.client.Do(req)
	if err != nil {
		return signResult{StatusCode: 0, Body: classifyTransportError("signin", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return signResult{StatusCode: resp.StatusCode, Body: "signin_read_failed"}
	}

	text := strings.TrimSpace(string(body))
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		payload = nil
	}

	return signResult{
		StatusCode: resp.StatusCode,
		Body:       text,
		Payload:    payload,
		OK:         resp.StatusCode == http.StatusOK || text == "has_signed_in_today" || isDigits(text),
	}
}

func (r *Runner) getSignInfo(ctx context.Context, accessToken string) map[string]any {
	headers := map[string]string{
		"Accept":        "application/json, text/plain, */*",
		"Authorization": "Bearer " + accessToken,
		"Origin":        "https://www.cc98.org",
		"Referer":       "https://www.cc98.org/",
		"User-Agent":    defaultUserAgent,
	}
	body, statusCode, err := get(ctx, r.client, webvpnBaseURL+defaultSignPath, headers)
	if err != nil || statusCode != http.StatusOK {
		return nil
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	return payload
}

func summarizeSignResult(sign signResult, signInfo map[string]any) signSummary {
	summary := signSummary{Status: "failed"}

	switch {
	case sign.Body == "has_signed_in_today":
		summary.Status = "already"
	case isDigits(sign.Body):
		summary.Status = "success"
		summary.Reward = intPtr(mustAtoi(sign.Body))
	case sign.Payload != nil:
		if boolValue(sign.Payload["hasSignedInToday"]) {
			summary.Status = "already"
		}
		if reward, ok := parseOptionalInt(sign.Payload["reward"]); ok {
			summary.Reward = intPtr(reward)
		} else if reward, ok := parseOptionalInt(sign.Payload["lastReward"]); ok {
			summary.Reward = intPtr(reward)
		}
		if streak, ok := parseOptionalInt(sign.Payload["signInCount"]); ok {
			summary.Streak = intPtr(streak)
		} else if streak, ok := parseOptionalInt(sign.Payload["lastSignInCount"]); ok {
			summary.Streak = intPtr(streak)
		}
	}

	if signInfo != nil {
		if summary.Reward == nil {
			if reward, ok := parseOptionalInt(signInfo["lastReward"]); ok {
				summary.Reward = intPtr(reward)
			}
		}
		if summary.Streak == nil {
			if streak, ok := parseOptionalInt(signInfo["lastSignInCount"]); ok {
				summary.Streak = intPtr(streak)
			}
		}
		if summary.Status == "failed" && boolValue(signInfo["hasSignedInToday"]) {
			summary.Status = "already"
		}
	}

	return summary
}

func formatResultText(summary signSummary, sign signResult) string {
	switch summary.Status {
	case "success", "already":
		parts := []string{"✅ 签到成功"}
		if summary.Reward != nil {
			parts = append(parts, fmt.Sprintf("🎁 %d财富值", *summary.Reward))
		}
		if summary.Streak != nil {
			parts = append(parts, fmt.Sprintf("📅 连续 %d 天", *summary.Streak))
		}
		return strings.Join(parts, " · ")
	default:
		return formatFailureText("签到失败", sign.Body)
	}
}

func formatFailureText(prefix, reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "请求失败"
	}
	if isWebVPNLoginText(reason) {
		reason = "登录状态失效"
	}
	return fmt.Sprintf("❌ %s · %s", prefix, reason)
}

func formatOutputLines(results []accountResult) []string {
	lines := make([]string, 0, len(results))
	for _, result := range results {
		lines = append(lines, fmt.Sprintf("账号%d %s", result.Index, result.ResultText))
	}
	return lines
}

func newHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout:   defaultConnectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   defaultConnectTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   defaultRequestTimeout,
		Jar:       newCookieJar(),
	}
}

func get(ctx context.Context, client *http.Client, targetURL string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, 0, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", defaultUserAgent)
	}

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

func parseErrorMessage(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return firstNonEmpty(stringValue(payload["error_description"]), stringValue(payload["error"]), stringValue(payload["message"]))
}

func isWebVPNLoginResponse(result tokenResult) bool {
	return isWebVPNLoginText(result.RawText)
}

func isWebVPNLoginText(text string) bool {
	markers := []string{`name="_csrf"`, "captcha_id", "wengine_vpn_ticket", "WebVPN"}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func boolValue(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

func parseOptionalInt(v any) (int, bool) {
	switch value := v.(type) {
	case float64:
		return int(value), true
	case int:
		return value, true
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return 0, false
		}
		num, err := strconv.Atoi(value)
		if err != nil {
			return 0, false
		}
		return num, true
	default:
		return 0, false
	}
}

func intPtr(v int) *int {
	return &v
}

func isDigits(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func mustAtoi(text string) int {
	value, _ := strconv.Atoi(text)
	return value
}

func stringValue(v any) string {
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value)
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func urlValues(pairs ...string) string {
	values := make([]string, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		values = append(values, fmt.Sprintf("%s=%s", url.QueryEscape(pairs[i]), url.QueryEscape(pairs[i+1])))
	}
	return strings.Join(values, "&")
}

func classifyTransportError(prefix string, err error) string {
	return fmt.Sprintf("%s 请求失败", prefix)
}

func unixNow() int64 {
	return time.Now().Unix()
}
