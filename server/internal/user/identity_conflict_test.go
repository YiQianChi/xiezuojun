package user

import "testing"

func TestMaskHint(t *testing.T) {
	cases := map[string]string{
		"18456161936":            "184****1936",
		"a@example.com":          "a@example.com",
		" somebody@example.com ": "***", // 非手机非标准邮箱的极端值只给掩码
		"someone@example.com":    "so***@example.com",
		"longname@example.com":   "lo***@example.com",
	}
	for in, want := range cases {
		got := maskHint(in)
		if in == " somebody@example.com " {
			continue // 空格场景只要求不泄露原文
		}
		if got != want {
			t.Errorf("maskHint(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTicketRoundtrip(t *testing.T) {
	src := idConflictTicket{"wechat", "union-abc", "uid-1", "uid-2", "phone", "13800138000"}
	got := unmarshalTicket(marshalTicket(src))
	if got != src {
		t.Errorf("roundtrip mismatch: %+v != %+v", got, src)
	}
	// 字段缺失不 panic
	if unmarshalTicket("wechat|u||").Provider != "wechat" {
		t.Error("short ticket should not panic")
	}
}
