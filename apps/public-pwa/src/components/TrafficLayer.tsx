import React, { useEffect } from 'react'
import { useMap } from '../contexts/MapContext'
import { incidentsApi } from '../api'

export function TrafficLayer() {
  const { map, trafficLayer } = useMap()

  useEffect(() => {
    if (!map) return

    // In production: fetch real-time traffic data from WebSocket or API
    // For now, mock data
    const mockCongestion = [
      { id: 'seg-1001', geometry: { type: 'LineString', coordinates: [[77.55, 12.92], [77.60, 12.95]] }, congestion: 2 },
      { id: 'seg-1002', geometry: { type: 'LineString', coordinates: [[77.65, 12.97], [77.70, 12.95]] }, congestion: 3 },
      { id: 'seg-2001', geometry: { type: 'LineString', coordinates: [[77.55, 12.97], [77.60, 12.97]] }, congestion: 4 },
    ]

    if (trafficLayer === 'congestion') {
      map.getSource('congestion')?.setData({
        type: 'FeatureCollection',
        features: mockCongestion.map((seg) => ({
          type: 'Feature',
          geometry: seg.geometry,
          properties: { congestion: seg.congestion, segment_id: seg.id },
        })),
      })

      // Add layer if not exists
      if (!map.getLayer('congestion-layer')) {
        map.addLayer({
          id: 'congestion-layer',
          type: 'line',
          source: 'congestion',
          paint: {
            'line-width': 6,
            'line-color': [
              'interpolate',
              ['linear'],
              ['get', 'congestion'],
              0, '#00c853', // green - free flow
              1, '#76ff03', // light green
              2, '#ffea00', // yellow - moderate
              3, '#ff9100', // orange - heavy
              4, '#ff3d00', // red - severe
              5, '#b71c1c', // dark red - gridlock
            ],
            'line-opacity': 0.8,
          },
        })
      }
    } else {
      if (map.getLayer('congestion-layer')) {
        map.removeLayer('congestion-layer')
      }
    }
  }, [map, trafficLayer])

  return null
}