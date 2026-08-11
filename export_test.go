package hyperex

// SetDebugModeForTest toggles vm debug checks for tests only.
func SetDebugModeForTest(enabled bool) (restore func()) {
	old := debugMode
	debugMode = enabled
	return func() { debugMode = old }
}
