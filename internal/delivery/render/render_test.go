package render

import "testing"

func TestFormatNumber(t *testing.T) {
	cases := []struct {
		input interface{}
		want  string
	}{
		{0, "0"},
		{2, "2"},
		{1000, "1.000"},
		{1234, "1.234"},
		{1234567, "1.234.567"},
		{int64(12345678), "12.345.678"},
		{float64(999.5), "999"},
		{-1234, "-1.234"},
		{"abc", "abc"},
	}
	for _, c := range cases {
		got := formatNumber(c.input)
		if got != c.want {
			t.Errorf("formatNumber(%v) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestFormatFloat(t *testing.T) {
	cases := []struct {
		input float64
		want  string
	}{
		{0, "0.00"},
		{450.00, "450.00"},
		{9999.99, "9999.99"},
		{12.5, "12.50"},
	}
	for _, c := range cases {
		got := formatFloat(c.input)
		if got != c.want {
			t.Errorf("formatFloat(%v) = %q, want %q", c.input, got, c.want)
		}
	}
}