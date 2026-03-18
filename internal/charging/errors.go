package charging

// ErrorCode identifies the category of a charging domain error.
type ErrorCode string

const (
	// CodeInvalidCallDirection is returned when a string cannot be parsed as a CallDirection.
	CodeInvalidCallDirection ErrorCode = "INVALID_CALL_DIRECTION"
	// CodeInvalidRateKey is returned when a string cannot be parsed as a RateKey.
	CodeInvalidRateKey ErrorCode = "INVALID_RATE_KEY"
	// CodeInvalidUnitType is returned when a string cannot be parsed as a UnitType.
	CodeInvalidUnitType ErrorCode = "INVALID_UNIT_TYPE"
)

// ChargingError is the typed error for all charging domain parse and validation failures.
// Callers should use errors.As to inspect the Code and act accordingly.
type ChargingError struct {
	Code    ErrorCode
	Message string
}

// Error implements the error interface.
func (e *ChargingError) Error() string {
	return string(e.Code) + ": " + e.Message
}

func newChargingError(code ErrorCode, msg string) *ChargingError {
	return &ChargingError{Code: code, Message: msg}
}

// NewInvalidCallDirection returns a ChargingError for an unrecognised call direction value.
func NewInvalidCallDirection(s string) *ChargingError {
	return newChargingError(CodeInvalidCallDirection, "invalid CallDirection: "+s)
}

// NewInvalidRateKey returns a ChargingError for a malformed or invalid rate key string.
func NewInvalidRateKey(msg string) *ChargingError {
	return newChargingError(CodeInvalidRateKey, msg)
}

// NewInvalidUnitType returns a ChargingError for an unrecognised unit type value.
func NewInvalidUnitType(s string) *ChargingError {
	return newChargingError(CodeInvalidUnitType, "invalid UnitType: "+s)
}
