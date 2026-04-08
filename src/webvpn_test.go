package main

import (
	"net/http"
	"testing"
)

func TestParseLoginPage(t *testing.T) {
	body := `
<input type="hidden" name="_csrf" value="csrf-token">
<input type="hidden" name="captcha_id" value="captcha-token">
<script>
  $("#login").click(function(){
    var password = encrypt($('input[name=password]').val(), "1234567890abcdef", "fedcba0987654321")
  })
</script>`

	fields, err := parseLoginPage(body)
	if err != nil {
		t.Fatalf("parseLoginPage returned error: %v", err)
	}
	if fields.CSRF != "csrf-token" || fields.CaptchaID != "captcha-token" {
		t.Fatalf("unexpected fields: %+v", fields)
	}
	if fields.PasswordKey != "1234567890abcdef" || fields.PasswordIV != "fedcba0987654321" {
		t.Fatalf("unexpected key/iv: %+v", fields)
	}
}

func TestRestoreCookieMapAddsDefaultRoute(t *testing.T) {
	jar := newCookieJar()
	restoreCookieMap(jar, map[string]string{
		"wengine_vpn_ticketwebvpn_zju_edu_cn": "ticket-value",
	})

	values := exportCookieMap(jar)
	if values["wengine_vpn_ticketwebvpn_zju_edu_cn"] != "ticket-value" {
		t.Fatalf("missing ticket cookie: %+v", values)
	}
	if values["route"] != defaultRouteValue {
		t.Fatalf("unexpected route cookie: %+v", values)
	}
}

func TestHasCoreWebVPNCookies(t *testing.T) {
	jar := newCookieJar()
	if hasCoreWebVPNCookies(jar) {
		t.Fatal("expected empty jar to be invalid")
	}
	jar.SetCookies(webvpnURL(), []*http.Cookie{{
		Name:  "wengine_vpn_ticketwebvpn_zju_edu_cn",
		Value: "ticket-value",
		Path:  "/",
	}})
	if !hasCoreWebVPNCookies(jar) {
		t.Fatal("expected jar with ticket to be valid")
	}
}
