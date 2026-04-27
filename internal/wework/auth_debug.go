package wework

import (
	"fmt"
	"os"
)

func authDebugf(format string, args ...any) {
	if os.Getenv("WEWORK_DEBUG_AUTH") == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "[wework-auth] "+format+"\n", args...)
}
