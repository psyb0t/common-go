package utils

import (
	"math/rand/v2"
	"strconv"
)

// GetRandomHex generates a random hexadecimal string of specified length with zero padding.
func GetRandomHex(length int) string {
	maxVal := 1

	for range length {
		maxVal *= 16
	}

	hexStr := strconv.FormatInt(int64(GetInsecureRandIntN(maxVal)), 16)
	// Pad with leading zeros to ensure exact length
	for len(hexStr) < length {
		hexStr = "0" + hexStr
	}

	return hexStr
}

// GetRandomAlphanumeric generates a random alphanumeric string of specified length.
func GetRandomAlphanumeric(length int) string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := range result {
		result[i] = chars[GetInsecureRandIntN(len(chars))]
	}

	return string(result)
}

// GetRandomString generates a random string with mixed case letters, numbers, and spaces.
func GetRandomString(length int) string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 "
	result := make([]byte, length)

	for i := range result {
		result[i] = chars[GetInsecureRandIntN(len(chars))]
	}

	return string(result)
}

// GetRandomStringInRange generates a random string with length between min and max.
func GetRandomStringInRange(minLength, maxLength int) string {
	length := GetInsecureRandIntN(maxLength-minLength+1) + minLength

	return GetRandomString(length)
}

// GetInsecureRandIntN returns a random int in range [0, n) using insecure math/rand.
func GetInsecureRandIntN(n int) int {
	return rand.IntN(n) //nolint:gosec
}
