package model

import (
	"go-ocs/internal/charging"
	"go-ocs/internal/common"
	"time"
)

type ClassificationPlan struct {
	RuleSetId string `json:"ruleSetId" yaml:"ruleSetId"`

	RuleSetName string `json:"ruleSetName" yaml:"ruleSetName"`

	UseServiceWindows bool `json:"useServiceWindows" yaml:"useServiceWindows"`

	DefaultServiceWindow string `json:"defaultServiceWindow" yaml:"defaultServiceWindow"`

	DefaultSourceType string `json:"defaultSourceType" yaml:"defaultSourceType"`

	ServiceWindows map[string]ServiceWindow `json:"serviceWindows" yaml:"serviceWindows"`

	ServiceTypes []ServiceType `json:"serviceTypes" yaml:"serviceTypes"`
}

type ServiceWindow struct {
	StartTime common.LocalTime `json:"startTime" yaml:"startTime"`
	EndTime   common.LocalTime `json:"endTime" yaml:"endTime"`
}

func (sw ServiceWindow) Duration() time.Duration {
	return sw.StartTime.Duration(sw.EndTime)
}

func (sw ServiceWindow) IsWithin(t time.Time) bool {
	return sw.StartTime.Before(t) && sw.EndTime.After(t)
}

type ServiceType struct {
	ServiceType string `json:"type" yaml:"type"`

	ChargingInformation ChargingInformation `json:"chargingInformation" yaml:"chargingInformation"`

	ServiceTypeRule string `json:"serviceTypeRule" yaml:"serviceTypeRule"`

	Description string `json:"description" yaml:"description"`

	SourceType string `json:"sourceType" yaml:"sourceType"`

	ServiceDirection string `json:"serviceDirection" yaml:"serviceDirection"`

	ServiceCategory string `json:"serviceCategory" yaml:"serviceCategory"`

	ServiceIdentifier string `json:"serviceIdentifier" yaml:"serviceIdentifier"`

	DefaultServiceCategory string `json:"defaultServiceCategory" yaml:"defaultServiceCategory"`

	UnitType charging.UnitType `json:"unitType" yaml:"unitType"`

	ServiceWindows []string `json:"serviceWindows" yaml:"serviceWindows"`

	ServiceWindowMap map[string]struct{}

	ServiceCategoryMap map[string]string `json:"serviceCategoryMap" yaml:"serviceCategoryMap"`
}
