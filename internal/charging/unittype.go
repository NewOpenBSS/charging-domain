package charging


// UnitType represents the type of charging unit.
type UnitType string

const (
	SECONDS  UnitType = "SECONDS"
	OCTETS   UnitType = "OCTETS"
	UNITS    UnitType = "UNITS"
	MONETARY UnitType = "MONETARY"
)

// descMap holds the description for each UnitType.
var descMap = map[UnitType]string{
	SECONDS:  "Seconds",
	OCTETS:   "Data",
	UNITS:    "Units",
	MONETARY: "Value",
}

// Description returns the human readable description of the UnitType.
func (u UnitType) Description() string {
	if d, ok := descMap[u]; ok {
		return d
	}
	return "Unknown"
}

// ParseUnitType validates and converts a string into a UnitType.
func ParseUnitType(s string) (UnitType, error) {
	u := UnitType(s)
	if _, ok := descMap[u]; !ok {
		return "", NewInvalidUnitType(s)
	}
	return u, nil
}
