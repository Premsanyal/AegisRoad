import React, { createContext, useContext, useState, useCallback, ReactNode } from 'react'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'

interface MapContextType {
  map: maplibregl.Map | null
  setMap: (map: maplibregl.Map) => void
  selectedLocation: GeoJSON.Position | null
  setSelectedLocation: (loc: GeoJSON.Position | null) => void
  bottomSheetContent: ReactNode | null
  setBottomSheetContent: (content: ReactNode | null) => void
  routeData: RouteData | null
  setRouteData: (data: RouteData | null) => void
  trafficLayer: 'congestion' | 'incidents' | 'cameras' | null
  setTrafficLayer: (layer: 'congestion' | 'incidents' | 'cameras' | null) => void
}

interface RouteData {
  primary: GeoJSON.LineString
  alternatives: GeoJSON.LineString[]
  eta: number
  distance: number
}

const MapContext = createContext<MapContextType | undefined>(undefined)

export function MapProvider({ children }: { children: ReactNode }) {
  const [map, setMap] = useState<maplibregl.Map | null>(null)
  const [selectedLocation, setSelectedLocation] = useState<GeoJSON.Position | null>(null)
  const [bottomSheetContent, setBottomSheetContent] = useState<ReactNode | null>(null)
  const [routeData, setRouteData] = useState<RouteData | null>(null)
  const [trafficLayer, setTrafficLayer] = useState<'congestion' | 'incidents' | 'cameras' | null>('congestion')

  const clearRoute = useCallback(() => {
    setRouteData(null)
    if (map) {
      map.getSource('route-primary')?.setData({ type: 'FeatureCollection', features: [] })
      map.getSource('route-alt')?.setData({ type: 'FeatureCollection', features: [] })
    }
  }, [map])

  return (
    <MapContext.Provider
      value={{
        map,
        setMap,
        selectedLocation,
        setSelectedLocation,
        bottomSheetContent,
        setBottomSheetContent,
        routeData,
        setRouteData,
        trafficLayer,
        setTrafficLayer,
      }}
    >
      {children}
    </MapContext.Provider>
  )
}

export function useMap() {
  const context = useContext(MapContext)
  if (!context) throw new Error('useMap must be used within MapProvider')
  return context
}