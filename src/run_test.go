package main

import "testing"

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
