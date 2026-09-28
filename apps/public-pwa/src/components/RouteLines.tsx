import React, { useEffect } from 'react'
import { useMap } from '../contexts/MapContext'

export function RouteLines() {
  const { map, routeData } = useMap()

  useEffect(() => {
    if (!map) return

    if (routeData) {
      // Primary route (reserved for emergency - show as dashed for public)
      map.getSource('route-primary')?.setData({
        type: 'Feature',
        geometry: routeData.primary,
      })

      // Alternative routes
      map.getSource('route-alt')?.setData({
        type: 'FeatureCollection',
        features: routeData.alternatives.map((alt, i) => ({
          type: 'Feature',
          geometry: alt,
          properties: { index: i },
        })),
      })
    } else {
      // Clear routes
      map.getSource('route-primary')?.setData({ type: 'FeatureCollection', features: [] })
      map.getSource('route-alt')?.setData({ type: 'FeatureCollection', features: [] })
    }
  }, [map, routeData])

  return null
}