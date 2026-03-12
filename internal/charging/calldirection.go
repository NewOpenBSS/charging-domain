package charging

import "fmt"

type CallDirection string

const (
	MO  CallDirection = "MO"
	MF  CallDirection = "MF"
	MT  CallDirection = "MT"
	ANY CallDirection = "*"
)

func (c CallDirection) String() string {
	return string(c)
}

func ParseCallDirection(s string) (CallDirection, error) {
	switch CallDirection(s) {
	case MO, MF, MT, ANY:
		return CallDirection(s), nil
	default:
		return "", fmt.Errorf("invalid CallDirection: %s", s)
	}
}
