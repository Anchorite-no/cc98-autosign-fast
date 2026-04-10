package main

import (
	"testing"
	"time"
)

func TestFormatResultTextSuccess(t *testing.T) {
	reward := 1141
	streak := 30
	text := formatResultText(signSummary{
		Status: "success",
		Reward: &reward,
		Streak: &streak,
	}, signResult{})

	expected := "✅ 签到成功 · 🎁 1141财富值 · 📅 连续 30 天"
	if text != expected {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestFormatOutputLinesAlwaysKeepsAccountPrefix(t *testing.T) {
	lines := formatOutputLines([]accountResult{{
		Index:      1,
		Username:   "anchorite",
		ResultText: "✅ 签到成功",
	}})
	if len(lines) != 1 || lines[0] != "账号1(anchorite) ✅ 签到成功" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestIsWebVPNLoginResponse(t *testing.T) {
	result := tokenResult{RawText: `<input name="_csrf" value="abc"><input name="captcha_id" value="def">`}
	if !isWebVPNLoginResponse(result) {
		t.Fatal("expected login page marker to be detected")
	}
}

func TestFormatTimingLines(t *testing.T) {
	lines := formatTimingLines(
		[]timingEntry{{Label: "WebVPN · GET /login", Duration: 120 * time.Millisecond}},
		[]accountResult{{
			Index:    1,
			Username: "anchorite",
			Timings: []timingEntry{
				{Label: "POST connect/token", Duration: 40 * time.Millisecond},
				{Label: "POST me/signin", Duration: 20 * time.Millisecond},
			},
		}},
	)

	if len(lines) != 2 {
		t.Fatalf("unexpected timing line count: %#v", lines)
	}
	if lines[0] != "WebVPN · GET /login 0.12s" {
		t.Fatalf("unexpected webvpn timing line: %q", lines[0])
	}
	if lines[1] != "账号1(anchorite)耗时 · POST connect/token 0.04s · POST me/signin 0.02s" {
		t.Fatalf("unexpected account timing line: %q", lines[1])
	}
}
