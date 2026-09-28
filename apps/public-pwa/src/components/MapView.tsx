import React, { useEffect, useRef } from 'react'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import { useMap } from '../contexts/MapContext'
import { routesApi } from '../api'
import { TrafficLayer } from './TrafficLayer'
import { IncidentMarkers } from './IncidentMarkers'
import { RouteLines } from './RouteLines'
import { LocationMarker } from './LocationMarker'

const BENGALURU_CENTER: [number, number] = [77.5946, 12.9716]
const TILE_URL = import.meta.env.VITE_TILE_URL || 'http://localhost:8080'

export function MapView() {
  const mapContainer = useRef<HTMLDivElement>(null)
  const { map, setMap, setSelectedLocation, trafficLayer, routeData } = useMap()

  useEffect(() => {
    if (map || !mapContainer.current) return

    const newMap = new maplibregl.Map({
      container: mapContainer.current,
      style: {
        version: 8,
        sources: {
          'openmaptiles': {
            type: 'vector',
            url: `${TILE_URL}/styles/openmaptiles.json`,
          },
        },
        layers: [
          {
            id: 'background',
            type: 'background',
            paint: { 'background-color': '#f5f5f5' },
          },
        ],
      },
      center: BENGALURU_CENTER,
      zoom: 12,
      pitch: 0,
      bearing: 0,
      antialias: true,
    })

    newMap.addControl(new maplibregl.NavigationControl(), 'top-right')
    newMap.addControl(new maplibregl.GeolocateControl({
      positionOptions: { enableHighAccuracy: true },
      trackUserLocation: true,
      showUserLocation: true,
    }), 'top-right')

    newMap.on('load', () => {
      // Add traffic layers after style loads
      setupTrafficLayers(newMap)
      setupRouteLayers(newMap)
    })

    newMap.on('click', (e) => {
      setSelectedLocation([e.lngLat.lng, e.lngLat.lat])
    })

    setMap(newMap)

    return () => {
      newMap.remove()
      setMap(null)
    }
  }, [map, setMap, setSelectedLocation, trafficLayer])

  const setupTrafficLayers = (map: maplibregl.Map) => {
    // Congestion layer from vector tiles or API
    if (!map.getSource('congestion')) {
      map.addSource('congestion', {
        type: 'geojson',
        data: { type: 'FeatureCollection', features: [] },
      })
    }
  }

  const setupRouteLayers = (map: maplibregl.Map) => {
    map.addSource('route-primary', {
      type: 'geojson',
      data: { type: 'FeatureCollection', features: [] },
    })
    map.addLayer({
      id: 'route-primary',
      type: 'line',
      source: 'route-primary',
      layout: { 'line-join': 'round', 'line-cap': 'round' },
      paint: {
        'line-color': '#1a73e8',
        'line-width': 5,
        'line-opacity': 0.9,
      },
    })

    map.addSource('route-alt', {
      type: 'geojson',
      data: { type: 'FeatureCollection', features: [] },
    })
    map.addLayer({
      id: 'route-alt',
      type: 'line',
      source: 'route-alt',
      layout: { 'line-join': 'round', 'line-cap': 'round' },
      paint: {
        'line-color': '#34a853',
        'line-width': 4,
        'line-opacity': 0.7,
        'line-dasharray': [8, 4],
      },
    })
  }

  return (
    <div className="map-view" ref={mapContainer}>
      <TrafficLayer />
      <IncidentMarkers />
      <RouteLines />
      {useMap().selectedLocation && <LocationMarker />}
    </div>
  )
}