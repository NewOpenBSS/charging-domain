package model

type ChargingInformation string

const (
	IMS ChargingInformation = "IMS"
	SMS ChargingInformation = "SMS"
	PDU ChargingInformation = "PDU"
	NEF ChargingInformation = "NEF"
)

func (c ChargingInformation) String() string {
	return string(c)
}
