import React, { useEffect, useState } from 'react'
import maplibregl from 'maplibre-gl'
import { useMap } from '../contexts/MapContext'
import { incidentsApi, CreateIncidentRequest } from '../api'
import { MapPin, AlertTriangle, AlertCircle } from 'lucide-react'

const INCIDENT_ICONS = {
  accident: AlertTriangle,
  breakdown: AlertCircle,
  construction: MapPin,
  flood: AlertTriangle,
  event: MapPin,
  hazard: AlertTriangle,
}

const INCIDENT_COLORS = {
  accident: '#ff3d00',
  breakdown: '#ff9100',
  construction: '#1a73e8',
  flood: '#2979ff',
  event: '#6a1b9a',
  hazard: '#d50000',
}

export function IncidentMarkers() {
  const { map } = useMap()
  const [incidents, setIncidents] = useState<any[]>([])
  const [markers, setMarkers] = useState<maplibregl.Marker[]>([])

  useEffect(() => {
    if (!map) return

    // Fetch active incidents
    incidentsApi.list({ status: 'active' }).then((data) => {
      setIncidents(data.incidents || [])
    }).catch(() => {
      // Mock data for development
      setIncidents([
        { id: 'inc-1', type: 'accident', severity: 3, geometry: { coordinates: [77.62, 12.91] }, description: 'Multi-vehicle accident' },
        { id: 'inc-2', type: 'construction', severity: 2, geometry: { coordinates: [77.65, 12.93] }, description: 'Road work in progress' },
        { id: 'inc-3', type: 'breakdown', severity: 1, geometry: { coordinates: [77.59, 12.95] }, description: 'Vehicle breakdown' },
      ])
    })

    return () => {
      markers.forEach((m) => m.remove())
    }
  }, [map])

  useEffect(() => {
    if (!map) return

    // Clear old markers
    markers.forEach((m) => m.remove())

    const newMarkers = incidents.map((incident) => {
      const [lng, lat] = incident.geometry.coordinates
      const Icon = INCIDENT_ICONS[incident.type as keyof typeof INCIDENT_ICONS] || MapPin
      const color = INCIDENT_COLORS[incident.type as keyof typeof INCIDENT_COLORS] || '#1a73e8'

      const el = document.createElement('div')
      el.className = 'incident-marker'
      el.innerHTML = `
        <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="2">
          <circle cx="12" cy="12" r="10" fill="${color}" opacity="0.2"/>
        </svg>
      `
      el.style.cursor = 'pointer'

      const marker = new maplibregl.Marker({ element: el, anchor: 'bottom' })
        .setLngLat([lng, lat])
        .setPopup(new maplibregl.Popup({ offset: 25 }).setHTML(`
          <div class="incident-popup">
            <h4>${incident.type.charAt(0).toUpperCase() + incident.type.slice(1)}</h4>
            <p>Severity: ${incident.severity}/5</p>
            <p>${incident.description}</p>
          </div>
        `))
        .addTo(map)

      return marker
    })

    setMarkers(newMarkers)
  }, [map, incidents])

  return null
}