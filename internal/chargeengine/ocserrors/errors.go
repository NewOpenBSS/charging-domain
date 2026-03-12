package ocserrors

import "go-ocs/internal/nchf"

type Code string

const (
	CodeGeneralError            Code = "GENERAL_ERROR"
	CodeInvalidReference        Code = "INVALID_REFERENCE"
	CodeUsedMoreThanGranted     Code = "USED_MORE_THAN_GRANTED"
	CodeNoGrantsFound           Code = "NO_GRANTS_FOUND"
	CodeRetransmit              Code = "RETRANSMIT"
	CodeServiceBarred           Code = "SERVICE_BARRED"
	CodeUnknownSubscriber       Code = "UNKNOWN_SUBSCRIBER"
	CodeSubscriberInactive      Code = "SUBSCRIBER_INACTIVE"
	CodeUnabledToClassification Code = "UNABLE_TO_CLASSIFY"
	CodeOutOfFunds              Code = "OUT_OF_FUNDS"
	CodeNoRatingEntry           Code = "NO_RATING_ENTRY"
	CodeUnableToClassify        Code = "UNABLE_TO_CLASSIFY"
	CodeUnknownCarrier          Code = "UNKNOWN_CARRIER"
)

type OcsError struct {
	Code    Code
	Message string
	Details any
}

func (e *OcsError) Error() string {
	return string(e.Code) + ": " + e.Message
}

type RetransmitError struct {
	code     Code
	Message  string
	Response *nchf.ChargingDataResponse
}

func (e *RetransmitError) Error() string {
	return string(e.code) + ": " + e.Message
}

func (e *RetransmitError) GetResponse() *nchf.ChargingDataResponse {
	return e.Response
}

func newError(code Code, msg string) *OcsError {
	return &OcsError{
		Code:    code,
		Message: msg,
	}
}

func CreateRetransmit(resp *nchf.ChargingDataResponse) *RetransmitError {
	return &RetransmitError{
		code:     CodeRetransmit,
		Response: resp,
		Message:  "Retransmitting session",
	}
}

func CreateGeneralError(msg string) *OcsError {
	return newError(CodeGeneralError, msg)
}

func CreateOutOfFunds(msg string) *OcsError {
	return newError(CodeOutOfFunds, msg)
}

func CreateUnknownSubscriber(msg string) *OcsError {
	return newError(CodeUnknownSubscriber, msg)
}

func CreateUnknownCarrier(msg string) *OcsError {
	return newError(CodeUnknownCarrier, msg)
}

func CreateClassificationError(msg string) *OcsError {
	return newError(CodeUnabledToClassification, msg)
}

func CreateNoRatingEntry(msg string) *OcsError {
	return newError(CodeNoRatingEntry, msg)
}

func CreateServiceBarred(msg string) *OcsError {
	return newError(CodeServiceBarred, msg)
}

func CreateUsedMoreThanGranted(msg string) *OcsError {
	return newError(CodeUsedMoreThanGranted, msg)
}

func CreateInvalidReferenced(msg string) *OcsError {
	return newError(CodeInvalidReference, msg)
}
