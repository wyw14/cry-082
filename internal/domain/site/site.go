package site

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidSite = errors.New("invalid site")
	ErrInvalidZone = errors.New("invalid zone")
)

type Site struct {
	ID              string
	Name            string
	Timezone        string
	ResponsibleUnit string
	CreatedAt       time.Time
	Version         int64
}

func New(id, name, timezone, responsibleUnit string, now time.Time) (Site, error) {
	id = strings.TrimSpace(id)
	name = strings.TrimSpace(name)
	responsibleUnit = strings.TrimSpace(responsibleUnit)
	if id == "" || name == "" || responsibleUnit == "" {
		return Site{}, ErrInvalidSite
	}
	if _, err := time.LoadLocation(timezone); err != nil {
		return Site{}, ErrInvalidSite
	}
	return Site{ID: id, Name: name, Timezone: timezone, ResponsibleUnit: responsibleUnit, CreatedAt: now.UTC(), Version: 1}, nil
}

type Zone struct {
	ID      string
	SiteID  string
	Name    string
	Purpose string
	Version int64
}

func NewZone(id, siteID, name, purpose string) (Zone, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(siteID) == "" || strings.TrimSpace(name) == "" {
		return Zone{}, ErrInvalidZone
	}
	return Zone{ID: id, SiteID: siteID, Name: name, Purpose: strings.TrimSpace(purpose), Version: 1}, nil
}

type MonitoringPoint struct {
	ID        string
	SiteID    string
	ZoneID    string
	Name      string
	Longitude float64
	Latitude  float64
	Active    bool
	Version   int64
}

func NewMonitoringPoint(id, siteID, zoneID, name string, longitude, latitude float64) (MonitoringPoint, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(siteID) == "" || strings.TrimSpace(zoneID) == "" || strings.TrimSpace(name) == "" {
		return MonitoringPoint{}, ErrInvalidSite
	}
	if longitude < -180 || longitude > 180 || latitude < -90 || latitude > 90 {
		return MonitoringPoint{}, ErrInvalidSite
	}
	return MonitoringPoint{ID: id, SiteID: siteID, ZoneID: zoneID, Name: name, Longitude: longitude, Latitude: latitude, Active: true, Version: 1}, nil
}
