import React, { useEffect, useRef } from 'react'
import { X, GripVertical } from 'lucide-react'

interface BottomSheetProps {
  content: React.ReactNode | null
  location: [number, number] | null
  onClose: () => void
}

export function BottomSheet({ content, location, onClose }: BottomSheetProps) {
  const sheetRef = useRef<HTMLDivElement>(null)
  const [height, setHeight] = React.useState(0)
  const [isOpen, setIsOpen] = React.useState(false)

  useEffect(() => {
    if (content) {
      setIsOpen(true)
      // Measure content height after render
      setTimeout(() => {
        if (sheetRef.current) {
          setHeight(Math.min(sheetRef.current.scrollHeight, window.innerHeight * 0.75))
        }
      }, 0)
    } else {
      setIsOpen(false)
    }
  }, [content])

  if (!isOpen) return null

  return (
    <div className="bottom-sheet-overlay" onClick={onClose}>
      <div
        ref={sheetRef}
        className="bottom-sheet"
        style={{ height: `${height}px` }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="sheet-handle" onClick={onClose}>
          <GripVertical size={24} />
        </div>
        <button className="close-btn" onClick={onClose}>
          <X size={20} />
        </button>
        <div className="sheet-content">
          {content}
        </div>
      </div>
    </div>
  )
}