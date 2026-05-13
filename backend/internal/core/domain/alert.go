package domain

import "time"

// ---------------------------------------------------------------------------------------------------------
//                                          DOMAIN LAYER
// ---------------------------------------------------------------------------------------------------------

// AlertType je typ upozornění, který backend posílá
type AlertType string

// Alert je odlišná business entity od TelemetryData.
// Telemetrie = "co se děje teď"
// Alert = "nastala vyjímečná situace, někdo musí jednat"
// Obě entity mají jiný životní cyklus, jiné consumers (telemetrii čte dashboard, alerty čte on-call tým)
const (
	AlertTemperatureMax AlertType = "temperature_exceeded_max_limit"
	AlertTemperatureMin AlertType = "temperature_exceeded_min_limit"
	AlertHumidityMax    AlertType = "humidity_exceeded_max_limit"
	AlertUnlocked       AlertType = "container_unlocked"
	AlertMaintenance    AlertType = "maintenance_required"
)

// Alert reprezentuje jeden alert poslaný backendem
// Message je simple string, protože chceme lidsky čitelnou zprávu
// (např. "Container #12: temperature 70°C exceeded max 40°C)
// Tu zprávu sestaví Processor Service při vytváření alertu - bude mít kontext (konkrétní hodnoty).
// Domain Layer ji jen nese.
type Alert struct {
	ID        string
	AssetID   string
	Type      AlertType
	Message   string
	CreatedAt time.Time
}
