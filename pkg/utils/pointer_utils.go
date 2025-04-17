package utils

// SafeDeref safely dereferences a string pointer and returns empty string if nil
func SafeDeref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
