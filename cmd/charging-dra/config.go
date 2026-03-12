// Package main provides configuration loading and validation for the go-ocs service.
// It loads environment-based settings and merges them with a Diameter YAML configuration file.
package main

import (
	"fmt"
	"go-ocs/internal/baseconfig"
	"time"
)

import _ "embed"

// Config represents the top-level application configuration loaded from
// environment variables and supplemented by Diameter YAML configuration.
type Config struct {
	Base        baseconfig.BaseConfig `yaml:"base"`
	ChargingDRA DraConfig             `yaml:"dra"`
	Diameter    DiameterFile          `yaml:"diameter"`
}

type DraConfig struct {
	// LastRequestWindow defines the duration window for tracking duplicate or recent requests.
	LastRequestWindow time.Duration `yaml:"lastRequestWindow"`

	// NationalDialCode is the default country code used when normalising MSISDN values.
	NationalDialCode string `yaml:"nationalDialCode"`

	// CacheRetention defines how long cached session or request data is retained.
	CacheRetention time.Duration `yaml:"cacheRetention"`

	// MaxIdleConns sets the maximum number of idle HTTP connections across all hosts.
	MaxIdleConns int `yaml:"maxIdleConns"`

	// MaxIdleConnsPerHost sets the maximum number of idle HTTP connections per host.
	MaxIdleConnsPerHost int `yaml:"maxIdleConnsPerHost"`

	// IdleConnTimeout defines how long an idle HTTP connection is kept alive.
	IdleConnTimeout time.Duration `yaml:"idleConnTimeout"`

	// DiameterConfigPath optionally specifies the path to an external Diameter YAML configuration file.
	DiameterConfigPath string `yaml:"diameterConfigPath"`
}

// DiameterFile represents the structured Diameter configuration as defined in YAML.
type DiameterFile struct {
	Transport TransportConfig `yaml:"transport"`
	LocalPeer LocalPeerConfig `yaml:"localPeer"`
	Watchdog  WatchdogConfig  `yaml:"watchdog"`
}

// TransportConfig defines transport-level settings for the Diameter server.
type TransportConfig struct {
	Network       string   `yaml:"network"` // tcp
	BindAddresses []string `yaml:"bindAddresses"`
	TLSCert       string   `yaml:"tlsCert"`
	TLSKey        string   `yaml:"tlsKey"`
}

// LocalPeerConfig defines the identity and capabilities of the local Diameter peer.
type LocalPeerConfig struct {
	Host             string `yaml:"host"`
	Realm            string `yaml:"realm"`
	VendorID         int    `yaml:"vendorId"`
	ProductName      string `yaml:"productName"`
	OriginStateID    uint32 `yaml:"originStateId"`
	FirmwareRevision uint32 `yaml:"firmwareRevision"`
}

// WatchdogConfig defines Diameter watchdog behaviour and timing parameters.
type WatchdogConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
}

// validateDiameterFields performs basic validation and defaulting of Diameter configuration fields.
func validateDiameterFields(d *DiameterFile) error {
	if d.LocalPeer.Host == "" {
		return fmt.Errorf("diameter.localPeer.host is required")
	}
	if d.LocalPeer.Realm == "" {
		return fmt.Errorf("diameter.localPeer.realm is required")
	}
	if d.Transport.Network == "" {
		d.Transport.Network = "tcp"
	}
	if d.Transport.BindAddresses == nil {
		d.Transport.BindAddresses = []string{"127.0.0.0:3868"}
	}
	if d.Watchdog.Interval == 0 {
		d.Watchdog.Interval = 30 * time.Second
	}
	return nil
}
