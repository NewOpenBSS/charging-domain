package model

import "go-ocs/internal/charging"

type Classification struct {
	Ratekey  charging.RateKey
	UnitType charging.UnitType
}
