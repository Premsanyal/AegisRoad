import React from 'react'
import { Outlet } from 'react-router-dom'
import { X, Menu } from 'lucide-react'
import { useMap } from '../contexts/MapContext'

export function Sidebar({ children }: { children: React.ReactNode }) {
  const { setBottomSheetContent } = useMap()
  const [isOpen, setIsOpen] = React.useState(true)

  return (
    <div className="sidebar-container">
      <button
        className={`sidebar-toggle ${isOpen ? 'open' : 'closed'}`}
        onClick={() => setIsOpen(!isOpen)}
        aria-label={isOpen ? 'Close sidebar' : 'Open sidebar'}
      >
        {isOpen ? <X size={24} /> : <Menu size={24} />}
      </button>

      <aside className={`sidebar ${isOpen ? 'open' : 'closed'}`}>
        <div className="sidebar-content">
          {children}
        </div>
      </aside>

      {!isOpen && (
        <div className="sidebar-overlay" onClick={() => setIsOpen(true)} />
      )}
    </div>
  )
}