package app

import "testing"

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
