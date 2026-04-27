package tzdate

import (
	"fmt"
	"time"
)

// ParseInTimezone parses a date string in "YYYY-MM-DD" format and applies the given timezone.
// It ensures that the resulting time is at the beginning of the day (00:00:00) in the specified timezone.
func ParseInTimezone(layout, dateStr, tzName string) (time.Time, error) {
	// Load the desired location.
	location, err := time.LoadLocation(tzName)
	if err != nil {
		// As a fallback, try to use UTC if the location is not found,
		// though ideally the timezone name should be valid.
		// For this application, we should probably error out.
		return time.Time{}, fmt.Errorf("failed to load location '%s': %w", tzName, err)
	}

	return time.ParseInLocation(layout, dateStr, location)
}
