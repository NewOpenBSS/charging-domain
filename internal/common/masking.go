package common

import "strings"

// MaskMSISDN returns a partially masked MSISDN suitable for logging.
// The first 3 and last 3 characters are preserved; everything in between
// is replaced with "****". Short values (fewer than 7 characters) are
// masked entirely to avoid exposing the full number.
//
// Examples:
//
//	"027123456789" → "027****789"
//	"021123456"   → "021***456"
//	"12345"       → "***"
func MaskMSISDN(msisdn string) string {
	const visible = 3
	if len(msisdn) <= visible*2 {
		return "***"
	}
	return msisdn[:visible] + strings.Repeat("*", 4) + msisdn[len(msisdn)-visible:]
}
