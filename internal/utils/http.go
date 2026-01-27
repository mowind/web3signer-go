// Package utils provides common utility functions for internal packages.
//
// This package contains shared functionality that is used across
// multiple internal modules, including validation and HTTP utilities.
package utils

import (
	"net/http"
	"time"
)

// CreateTransport creates an HTTP transport with optimized connection pooling.
//
// The transport is configured for production use with connection pooling:
//   - MaxIdleConns: Maximum idle connections across all hosts
//   - MaxIdleConnsPerHost: Maximum idle connections per host
//   - IdleConnTimeout: Timeout for idle connections before closing
//   - ResponseHeaderTimeout: Timeout for receiving response headers
//
// Parameters:
//   - maxIdle: Maximum idle connections (default: 100)
//   - idleTimeout: Timeout for idle connections (default: 90s)
//
// Returns:
//   - *http.Transport: Configured HTTP transport instance
//
// Example:
//
//	transport := CreateTransport(100, 90*time.Second)
//	client := &http.Client{Transport: transport}
func CreateTransport(maxIdle int, idleTimeout time.Duration) *http.Transport {
	return &http.Transport{
		MaxIdleConns:          maxIdle,
		MaxIdleConnsPerHost:   maxIdle,
		IdleConnTimeout:       idleTimeout,
		ResponseHeaderTimeout: 10 * time.Second,
		DisableCompression:    false,
		DisableKeepAlives:     false,
	}
}
