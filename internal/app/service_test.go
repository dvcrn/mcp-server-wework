package app

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestDecodeBase64Loose(t *testing.T) {
	want := bytes.Repeat([]byte("The quick brown fox. "), 20)
	std := base64.StdEncoding.EncodeToString(want)

	// Simulate 76-column PEM-style wrapping with internal newlines.
	var wrapped string
	for i := 0; i < len(std); i += 76 {
		end := i + 76
		if end > len(std) {
			end = len(std)
		}
		wrapped += std[i:end] + "\n"
	}

	cases := map[string]string{
		"plain":            std,
		"wrapped":          wrapped,
		"leading/trailing": "  \n" + std + "\n  ",
		"spaces inside":    std[:10] + " " + std[10:],
		"url-safe":         base64.URLEncoding.EncodeToString(want),
		"raw (no padding)": base64.RawStdEncoding.EncodeToString(want),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := decodeBase64Loose(in)
			if err != nil {
				t.Fatalf("decodeBase64Loose: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("decoded mismatch for %s", name)
			}
		})
	}

	if _, err := decodeBase64Loose("not valid base64!!!"); err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestParseDateSelection(t *testing.T) {
	t.Run("single date", func(t *testing.T) {
		dates, err := parseDateSelection("2026-04-06")
		if err != nil {
			t.Fatalf("parseDateSelection returned error: %v", err)
		}
		if len(dates) != 1 || dates[0].Format("2006-01-02") != "2026-04-06" {
			t.Fatalf("unexpected dates: %#v", dates)
		}
	})

	t.Run("comma separated dates", func(t *testing.T) {
		dates, err := parseDateSelection("2026-04-06,2026-04-08")
		if err != nil {
			t.Fatalf("parseDateSelection returned error: %v", err)
		}
		if len(dates) != 2 {
			t.Fatalf("expected 2 dates, got %d", len(dates))
		}
	})

	t.Run("date range", func(t *testing.T) {
		dates, err := parseDateSelection("2026-04-06~2026-04-08")
		if err != nil {
			t.Fatalf("parseDateSelection returned error: %v", err)
		}
		if len(dates) != 3 {
			t.Fatalf("expected 3 dates, got %d", len(dates))
		}
		if dates[2].Format("2006-01-02") != "2026-04-08" {
			t.Fatalf("unexpected end date: %s", dates[2].Format("2006-01-02"))
		}
	})
}
