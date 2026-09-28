import { Routes, Route, Navigate } from 'react-router-dom'
import { Sidebar } from './components/Sidebar'
import { MapView } from './components/MapView'
import { RoutePanel } from './components/RoutePanel'
import { IncidentReport } from './components/IncidentReport'
import { Settings } from './components/Settings'
import { useAuth } from './contexts/AuthContext'
import { useMap } from './contexts/MapContext'
import { Header } from './components/Header'
import { BottomSheet } from './components/BottomSheet'

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth()
  if (loading) return <div className="loading">Loading...</div>
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  const { user } = useAuth()
  const { selectedLocation, setSelectedLocation, bottomSheetContent, setBottomSheetContent } = useMap()

  return (
    <div className="app">
      <Header />
      <div className="app-body">
        <MapView />
        <Sidebar>
          <Routes>
            <Route path="/" element={<PrivateRoute><RoutePanel /></PrivateRoute>} />
            <Route path="/report" element={<PrivateRoute><IncidentReport /></PrivateRoute>} />
            <Route path="/settings" element={<PrivateRoute><Settings /></PrivateRoute>} />
            <Route path="/login" element={<LoginPage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </Sidebar>
      </div>
      <BottomSheet
        content={bottomSheetContent}
        location={selectedLocation}
        onClose={() => setBottomSheetContent(null)}
      />
    </div>
  )
}

function LoginPage() {
  const { login } = useAuth()
  const [phone, setPhone] = React.useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await login(phone)
  }

  return (
    <div className="login-page">
      <div className="login-card">
        <h1>AegisRoad</h1>
        <p>Smart traffic navigation for Bengaluru</p>
        <form onSubmit={handleSubmit}>
          <input
            type="tel"
            placeholder="Enter phone number"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            required
          />
          <button type="submit">Continue</button>
        </form>
      </div>
    </div>
  )
}