import React, { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Search, Navigation, MapPin, Clock, AlertTriangle, ChevronDown, X } from 'lucide-react'
import { useMap } from '../contexts/MapContext'
import { routesApi, CreateRouteRequest } from '../api'

export function RoutePanel() {
  const navigate = useNavigate()
  const { setSelectedLocation, setBottomSheetContent, setRouteData } = useMap()

  const [origin, setOrigin] = useState<{ coords: [number, number]; label: string } | null>(null)
  const [destination, setDestination] = useState<{ coords: [number, number]; label: string } | null>(null)
  const [loading, setLoading] = useState(false)
  const [showHistory, setShowHistory] = useState(false)
  const [recentRoutes, setRecentRoutes] = useState<any[]>([])

  const handleSearch = async (type: 'origin' | 'destination') => {
    setBottomSheetContent(
      <LocationSearch
        onSelect={(loc) => {
          if (type === 'origin') setOrigin(loc)
          else setDestination(loc)
          setBottomSheetContent(null)
        }}
      />
    )
  }

  const handleSwap = () => {
    setOrigin(destination)
    setDestination(origin)
  }

  const handleGetRoute = async () => {
    if (!origin || !destination) return

    setLoading(true)
    try {
      const req: CreateRouteRequest = {
        origin: origin.coords,
        destination: destination.coords,
        return_alternatives: true,
        max_alternatives: 3,
        profile: 'car',
      }

      const response = await routesApi.create(req)

      // For public users, the "recommended_route" is the 2nd best (1st is reserved)
      const primaryRoute = response.recommended_route || response.primary_route
      const altRoutes = response.alternative_routes || []

      setRouteData({
        primary: primaryRoute.geometry,
        alternatives: altRoutes.map((r: any) => r.geometry),
        eta: primaryRoute.duration_with_traffic_seconds,
        distance: primaryRoute.distance_meters,
      })

      // Show bottom sheet with route options
      setBottomSheetContent(
        <RouteOptions
          primary={primaryRoute}
          alternatives={altRoutes}
          onStart={() => navigate('/navigation')} // Would start navigation
        />
      )
    } catch (error) {
      console.error('Route failed:', error)
      alert('Failed to calculate route. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="route-panel">
      <div className="search-card">
        <div className="search-field">
          <label>From</label>
          <button className="search-input" onClick={() => handleSearch('origin')}>
            <MapPin size={18} className="input-icon" />
            <span>{origin?.label || 'Current location'}</span>
            <X size={18} onClick={(e) => { e.stopPropagation(); setOrigin(null) }} />
          </button>
        </div>

        <button className="swap-btn" onClick={handleSwap} disabled={!origin || !destination}>
          <Navigation size={20} />
        </button>

        <div className="search-field">
          <label>To</label>
          <button className="search-input" onClick={() => handleSearch('destination')}>
            <MapPin size={18} className="input-icon" />
            <span>{destination?.label || 'Enter destination'}</span>
            <X size={18} onClick={(e) => { e.stopPropagation(); setDestination(null) }} />
          </button>
        </div>

        <button className="get-route-btn" onClick={handleGetRoute} disabled={loading || !destination}>
          {loading ? 'Calculating...' : 'Get Route'}
        </button>
      </div>

      <div className="recent-routes">
        <div className="section-header">
          <h3>Recent Routes</h3>
          <button className="toggle-history" onClick={() => setShowHistory(!showHistory)}>
            <ChevronDown size={16} className={showHistory ? 'rotated' : ''} />
          </button>
        </div>

        {showHistory && recentRoutes.length > 0 && (
          <div className="history-list">
            {recentRoutes.map((route) => (
              <button key={route.id} className="history-item" onClick={() => {}}>
                <span>{route.origin} → {route.destination}</span>
                <small>{formatTime(route.created_at)}</small>
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="quick-actions">
        <button className="action-btn" onClick={() => navigate('/report')}>
          <AlertTriangle size={20} />
          <span>Report Incident</span>
        </button>
        <button className="action-btn">
          <Clock size={20} />
          <span>Commute Times</span>
        </button>
      </div>
    </div>
  )
}

function LocationSearch({ onSelect }: { onSelect: (loc: { coords: [number, number]; label: string }) => void }) {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<any[]>([])
  const [loading, setLoading] = useState(false)

  React.useEffect(() => {
    if (query.length < 2) return
    setLoading(true)
    // In production: call geocoding API (Nominatim, Mapbox, etc.)
    // For now, mock results
    setTimeout(() => {
      setResults([
        { coords: [77.5946, 12.9716], label: 'MG Road, Bengaluru' },
        { coords: [77.62, 12.93], label: 'Koramangala 80ft Road' },
        { coords: [77.70, 12.96], label: 'Marathahalli' },
        { coords: [77.55, 12.97], label: 'Old Airport Road' },
        { coords: [77.65, 12.93], label: 'HSR Layout' },
      ].filter((r) => r.label.toLowerCase().includes(query.toLowerCase())))
      setLoading(false)
    }, 300)
  }, [query])

  return (
    <div className="location-search">
      <input
        type="text"
        placeholder="Search places..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        autoFocus
      />
      {loading && <div className="loading">Searching...</div>}
      <ul>
        {results.map((r) => (
          <li key={r.label} onClick={() => onSelect(r)}>
            <MapPin size={18} /> {r.label}
          </li>
        ))}
      </ul>
    </div>
  )
}

function RouteOptions({ primary, alternatives, onStart }: any) {
  const [selected, setSelected] = useState(primary)

  const formatDuration = (seconds: number) => {
    const mins = Math.round(seconds / 60)
    if (mins < 60) return `${mins} min`
    return `${Math.floor(mins / 60)}h ${mins % 60}min`
  }

  const formatDistance = (meters: number) => {
    if (meters < 1000) return `${meters}m`
    return `${(meters / 1000).toFixed(1)} km`
  }

  return (
    <div className="route-options">
      <h3>Route Options</h3>

      <div className="route-option recommended" onClick={() => setSelected(primary)}>
        <div className="route-badge">Recommended</div>
        <div className="route-info">
          <div className="route-eta">
            <Clock size={20} />
            <span>{formatDuration(selected.duration_with_traffic_seconds)}</span>
          </div>
          <div className="route-distance">
            <Navigation size={16} />
            <span>{formatDistance(selected.distance_meters)}</span>
          </div>
          <div className="route-congestion">
            Congestion: {['Free', 'Light', 'Moderate', 'Heavy', 'Severe', 'Gridlock'][selected.congestion_level]}
          </div>
        </div>
        {selected === primary && <span className="selected-indicator">✓</span>}
      </div>

      {alternatives.map((alt: any, i: number) => (
        <div key={i} className="route-option" onClick={() => setSelected(alt)}>
          <div className="route-info">
            <div className="route-eta">
              <Clock size={20} />
              <span>{formatDuration(alt.duration_with_traffic_seconds)}</span>
              <span className="diff">+{Math.round((alt.duration_with_traffic_seconds - primary.duration_with_traffic_seconds) / 60)} min</span>
            </div>
            <div className="route-distance">
              <Navigation size={16} />
              <span>{formatDistance(alt.distance_meters)}</span>
            </div>
          </div>
        </div>
      ))}

      <button className="start-nav-btn" onClick={onStart}>
        Start Navigation
      </button>
    </div>
  )
}

function formatTime(iso: string) {
  const date = new Date(iso)
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}