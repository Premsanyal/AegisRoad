import React, { useEffect } from 'react'
import maplibregl from 'maplibre-gl'
import { useMap } from '../contexts/MapContext'
import { MapPin } from 'lucide-react'

export function LocationMarker() {
  const { map, selectedLocation, setSelectedLocation, setBottomSheetContent } = useMap()
  const markerRef = useRef<maplibregl.Marker | null>(null)

  useEffect(() => {
    if (!map || !selectedLocation) {
      markerRef.current?.remove()
      markerRef.current = null
      return
    }

    const el = document.createElement('div')
    el.className = 'location-marker'
    el.innerHTML = `
      <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#1a73e8" stroke-width="2.5">
        <path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/>
        <circle cx="12" cy="10" r="3" fill="#1a73e8"/>
      </svg>
    `
    el.style.cursor = 'pointer'

    markerRef.current = new maplibregl.Marker({ element: el, anchor: 'bottom' })
      .setLngLat(selectedLocation)
      .addTo(map)

    // Fit map to show marker
    map.panTo(selectedLocation)

    return () => {
      markerRef.current?.remove()
    }
  }, [map, selectedLocation])

  return null
}

import { useRef } from 'react'