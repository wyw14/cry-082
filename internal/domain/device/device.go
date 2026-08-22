package device

import (
	"errors"
	"net"
	"strings"
	"time"
)

var (
	ErrInvalidDevice     = errors.New("invalid device")
	ErrInvalidTransition = errors.New("invalid device lifecycle transition")
)

type Status string

const (
	StatusRegistered  Status = "registered"
	StatusOnline      Status = "online"
	StatusOffline     Status = "offline"
	StatusMaintenance Status = "maintenance"
	StatusReplaced    Status = "replaced"
	StatusRetired     Status = "retired"
)

type NetworkConfig struct {
	Host     string
	Port     int
	Protocol string
}

func (n NetworkConfig) Valid() bool {
	if strings.TrimSpace(n.Host) == "" || n.Port < 1 || n.Port > 65535 {
		return false
	}
	if net.ParseIP(n.Host) == nil && strings.ContainsAny(n.Host, " /\\") {
		return false
	}
	return n.Protocol == "mqtt" || n.Protocol == "http" || n.Protocol == "modbus-tcp"
}

type Device struct {
	ID              string
	Code            string
	Model           string
	SiteID          string
	PointID         string
	InstallLocation string
	Network         NetworkConfig
	Status          Status
	LastSeenAt      *time.Time
	ReplacementID   string
	Version         int64
}

func New(id, code, model, siteID, pointID, location string, network NetworkConfig) (Device, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(code) == "" || strings.TrimSpace(model) == "" || strings.TrimSpace(siteID) == "" || strings.TrimSpace(pointID) == "" || !network.Valid() {
		return Device{}, ErrInvalidDevice
	}
	return Device{ID: id, Code: code, Model: model, SiteID: siteID, PointID: pointID, InstallLocation: location, Network: network, Status: StatusRegistered, Version: 1}, nil
}

func (d *Device) Transition(next Status, at time.Time, replacementID string) error {
	allowed := map[Status]map[Status]bool{
		StatusRegistered:  {StatusOnline: true, StatusMaintenance: true, StatusRetired: true},
		StatusOnline:      {StatusOffline: true, StatusMaintenance: true, StatusReplaced: true, StatusRetired: true},
		StatusOffline:     {StatusOnline: true, StatusMaintenance: true, StatusReplaced: true, StatusRetired: true},
		StatusMaintenance: {StatusOnline: true, StatusOffline: true, StatusReplaced: true, StatusRetired: true},
	}
	if !allowed[d.Status][next] {
		return ErrInvalidTransition
	}
	if next == StatusReplaced && strings.TrimSpace(replacementID) == "" {
		return ErrInvalidTransition
	}
	d.Status = next
	d.ReplacementID = replacementID
	d.Version++
	if next == StatusOnline {
		seen := at.UTC()
		d.LastSeenAt = &seen
	}
	return nil
}

func (d *Device) MarkSeen(at time.Time) error {
	if d.Status == StatusRetired || d.Status == StatusReplaced {
		return ErrInvalidTransition
	}
	seen := at.UTC()
	d.LastSeenAt = &seen
	if d.Status == StatusRegistered || d.Status == StatusOffline {
		d.Status = StatusOnline
	}
	d.Version++
	return nil
}
