import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';

const STORAGE_KEY = 'cgl_workshop_mode';

interface WorkshopModeData {
  workshopId: string;
  workshopName: string;
}

interface WorkshopModeContextType {
  /** True if staff/head has entered workshop mode */
  isInWorkshopMode: boolean;
  /** The workshop ID when in workshop mode */
  activeWorkshopId: string | null;
  /** The workshop name when in workshop mode */
  activeWorkshopName: string | null;
  /** Enter workshop mode for a specific workshop */
  enterWorkshopMode: (workshopId: string, workshopName: string) => void;
  /** Exit workshop mode and return to normal app behavior */
  exitWorkshopMode: () => void;
}

const WorkshopModeContext = createContext<WorkshopModeContextType | undefined>(undefined);

export function useWorkshopMode() {
  const context = useContext(WorkshopModeContext);
  if (!context) {
    throw new Error('useWorkshopMode must be used within WorkshopModeProvider');
  }
  return context;
}

interface WorkshopModeProviderProps {
  children: React.ReactNode;
}

export function WorkshopModeProvider({ children }: WorkshopModeProviderProps) {
  const [workshopData, setWorkshopData] = useState<WorkshopModeData | null>(() => {
    // Initialize from localStorage
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        return JSON.parse(stored) as WorkshopModeData;
      }
    } catch {
      // Ignore storage errors
    }
    return null;
  });

  // Persist to localStorage when workshop mode changes
  useEffect(() => {
    if (workshopData) {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(workshopData));
    } else {
      localStorage.removeItem(STORAGE_KEY);
    }
  }, [workshopData]);

  const enterWorkshopMode = useCallback((workshopId: string, workshopName: string) => {
    setWorkshopData({ workshopId, workshopName });
  }, []);

  const exitWorkshopMode = useCallback(() => {
    setWorkshopData(null);
  }, []);

  const value: WorkshopModeContextType = {
    isInWorkshopMode: workshopData !== null,
    activeWorkshopId: workshopData?.workshopId ?? null,
    activeWorkshopName: workshopData?.workshopName ?? null,
    enterWorkshopMode,
    exitWorkshopMode,
  };

  return (
    <WorkshopModeContext.Provider value={value}>
      {children}
    </WorkshopModeContext.Provider>
  );
}
