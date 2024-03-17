package rmqgath

func clampString(s string, maxSize int) string {
	if len(s) <= maxSize {
		return s
	}

	// Ensure we do not split in the middle of a UTF-8 character
	// by finding the boundary of the last complete character within the limit.
	truncLimit := maxSize - 3 // Reserve space for "..."
	for i := range s {
		if i > truncLimit {
			return s[:i] + "..."
		}
	}
	return s
}
