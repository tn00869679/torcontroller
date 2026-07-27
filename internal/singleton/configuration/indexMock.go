package configuration

// MockConfiguration provides a mock implementation of Configuration.
func MockConfiguration(minRead uint, minWrite uint) *Configuration {
	return &Configuration{
		RateLimit: RateLimitConfig{
			MinReadRate:  minRead,  // Example value for min read rate
			MinWriteRate: minWrite, // Example value for min write rate
		},
	}
}
