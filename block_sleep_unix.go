//go:build !windows

package simplesocksproxy

func PreventSleep() error {
	// No-op on non-Windows platforms
	return nil
}

func AllowSleep() error {
	// No-op on non-Windows platforms
	return nil
}
