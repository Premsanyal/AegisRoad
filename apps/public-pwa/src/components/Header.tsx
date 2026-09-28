import React from 'react'
import { Menu, Bell, Settings, LogOut, User, Shield } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { useMap } from '../contexts/MapContext'

export function Header() {
  const { user, logout } = useAuth()
  const { trafficLayer, setTrafficLayer } = useMap()
  const [showMenu, setShowMenu] = React.useState(false)

  const layers = [
    { id: 'congestion', label: 'Traffic', icon: '🚦' },
    { id: 'incidents', label: 'Incidents', icon: '⚠️' },
    { id: 'cameras', label: 'Cameras', icon: '📹' },
  ]

  return (
    <header className="app-header">
      <div className="header-left">
        <button className="menu-btn" onClick={() => setShowMenu(!showMenu)}>
          <Menu size={24} />
        </button>
        <div className="logo">
          <Shield size={20} />
          <span>AegisRoad</span>
        </div>
      </div>

      <div className="header-center">
        <div className="layer-tabs">
          {layers.map((layer) => (
            <button
              key={layer.id}
              className={`layer-tab ${trafficLayer === layer.id ? 'active' : ''}`}
              onClick={() => setTrafficLayer(layer.id as any)}
            >
              <span className="layer-icon">{layer.icon}</span>
              <span className="layer-label">{layer.label}</span>
            </button>
          ))}
        </div>
      </div>

      <div className="header-right">
        {user && (
          <>
            <button className="icon-btn" title="Alerts">
              <Bell size={22} />
            </button>
            <div className="user-menu">
              <button className="user-btn" onClick={() => setShowMenu(!showShowMenu)}>
                <User size={20} />
                <span>{user.name || user.phone}</span>
              </button>
              {showMenu && (
                <div className="user-dropdown">
                  <div className="user-info">
                    <User size={32} />
                    <div>
                      <strong>{user.name || 'User'}</strong>
                      <small>{user.role}</small>
                    </div>
                  </div>
                  <hr />
                  <button onClick={() => { setShowMenu(false); /* navigate to settings */ }}>
                    <Settings size={18} /> Settings
                  </button>
                  <button onClick={() => { logout(); setShowMenu(false); }}>
                    <LogOut size={18} /> Logout
                  </button>
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </header>
  )
}