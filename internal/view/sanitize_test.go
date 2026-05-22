package view

import "testing"

func TestSanitizeTerminal(t *testing.T) {
	// C1 control characters used in tests (built via \u escapes so the
	// source file stays free of raw control bytes).
	const (
		nel    = "" // C1 NEL
		csiC1  = "" // C1 CSI
		oscC1  = "" // C1 OSC
	)

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		{
			name: "plain ascii",
			in:   "hello world",
			want: "hello world",
		},
		{
			name: "tab newline cr preserved",
			in:   "a\tb\nc\rd",
			want: "a\tb\nc\rd",
		},
		{
			name: "OSC 8 hyperlink stripped, label preserved",
			in:   "\x1b]8;;file:///tmp/x\x1b\\Click\x1b]8;;\x1b\\",
			want: "Click",
		},
		{
			name: "OSC 8 hyperlink with BEL terminator",
			in:   "\x1b]8;;https://evil.example/\x07Click\x1b]8;;\x07",
			want: "Click",
		},
		{
			name: "OSC 52 clipboard write fully stripped",
			in:   "\x1b]52;c;ZXZpbA==\x07",
			want: "",
		},
		{
			name: "DECSET 1049 alt screen stripped",
			in:   "\x1b[?1049h",
			want: "",
		},
		{
			name: "plain SGR red passed through unchanged",
			in:   "\x1b[31mhello\x1b[0m",
			want: "\x1b[31mhello\x1b[0m",
		},
		{
			name: "SGR multi-param passed through",
			in:   "\x1b[1;38;5;242mfoo\x1b[m",
			want: "\x1b[1;38;5;242mfoo\x1b[m",
		},
		{
			name: "bare BEL stripped",
			in:   "hello\x07world",
			want: "helloworld",
		},
		{
			name: "DEL stripped",
			in:   "a\x7fb",
			want: "ab",
		},
		{
			name: "C0 controls stripped except tab/lf/cr",
			in:   "a\x01b\x02c\x08d\x0be\x0cf",
			want: "abcdef",
		},
		{
			name: "two-char ESC sequence (reverse index) stripped",
			in:   "before\x1bMafter",
			want: "beforeafter",
		},
		{
			name: "DCS sequence stripped",
			in:   "x\x1bPpayload\x1b\\y",
			want: "xy",
		},
		{
			name: "APC sequence stripped",
			in:   "x\x1b_payload\x1b\\y",
			want: "xy",
		},
		{
			name: "PM sequence stripped",
			in:   "x\x1b^payload\x1b\\y",
			want: "xy",
		},
		{
			name: "non-SGR CSI stripped (cursor up)",
			in:   "a\x1b[2Ab",
			want: "ab",
		},
		{
			name: "lone trailing ESC dropped",
			in:   "abc\x1b",
			want: "abc",
		},
		{
			name: "utf8 multibyte preserved",
			in:   "héllo 中文",
			want: "héllo 中文",
		},
		{
			name: "utf8 with embedded OSC stripped",
			in:   "café\x1b]52;c;ZGF0YQ==\x07ok",
			want: "caféok",
		},
		{
			name: "C1 NEL stripped",
			in:   "a" + nel + "b",
			want: "ab",
		},
		{
			name: "C1 CSI stripped",
			in:   "a" + csiC1 + "b",
			want: "ab",
		},
		{
			name: "C1 OSC stripped",
			in:   "a" + oscC1 + "b",
			want: "ab",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeTerminal(tc.in)
			if got != tc.want {
				t.Errorf("SanitizeTerminal(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
