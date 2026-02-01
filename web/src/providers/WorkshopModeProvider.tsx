import React, { createContext, useCallback, useContext } from "react";
import { useAuth } from "./AuthProvider";
import { useSetActiveWorkshop } from "@/api/hooks";

interface WorkshopModeContextType {
  /** True if staff/head/individual has entered workshop mode */
  isInWorkshopMode: boolean;
  /** The workshop ID when in workshop mode */
  activeWorkshopId: string | null;
  /** The workshop name when in workshop mode */
  activeWorkshopName: string | null;
  /** Enter workshop mode for a specific workshop (calls backend API) */
  enterWorkshopMode: (
    workshopId: string,
    workshopName: string,
  ) => Promise<void>;
  /** Exit workshop mode and return to normal app behavior (calls backend API) */
  exitWorkshopMode: () => Promise<void>;
  /** True if the enter/exit operation is in progress */
  isLoading: boolean;
}

const WorkshopModeContext = createContext<WorkshopModeContextType | undefined>(
  undefined,
);

export function useWorkshopMode() {
  const context = useContext(WorkshopModeContext);
  if (!context) {
    throw new Error("useWorkshopMode must be used within WorkshopModeProvider");
  }
  return context;
}

interface WorkshopModeProviderProps {
  children: React.ReactNode;
}

export function WorkshopModeProvider({ children }: WorkshopModeProviderProps) {
  const { backendUser, isParticipant, retryBackendFetch } = useAuth();
  const setActiveWorkshop = useSetActiveWorkshop();

  // Workshop mode is derived from backend user data
  // For head/staff/individual: backendUser.role.workshop is set when in workshop mode (can leave)
  // For participants: backendUser.role.workshop is always their assigned workshop (can't leave)
  // isInWorkshopMode only applies to non-participants who chose to enter
  const workshop = backendUser?.role?.workshop;
  const hasWorkshop = workshop !== undefined && workshop !== null;
  // Participants are always in their workshop but NOT in "workshop mode" (they can't exit)
  const isInWorkshopMode = hasWorkshop && !isParticipant;
  const activeWorkshopId = workshop?.id ?? null;
  const activeWorkshopName = workshop?.name ?? null;

  const enterWorkshopMode = useCallback(
    async (workshopId: string, _workshopName: string) => {
      // Skip if already in this workshop or mutation is pending
      if (activeWorkshopId === workshopId || setActiveWorkshop.isPending) {
        return;
      }
      // Call backend API to set active workshop
      await setActiveWorkshop.mutateAsync(workshopId);
      // Refetch backend user to get updated workshop context
      retryBackendFetch();
    },
    [activeWorkshopId, setActiveWorkshop, retryBackendFetch],
  );

  const exitWorkshopMode = useCallback(async () => {
    // Skip if not in workshop mode or mutation is pending
    if (!isInWorkshopMode || setActiveWorkshop.isPending) {
      return;
    }
    // Call backend API to clear active workshop
    await setActiveWorkshop.mutateAsync(null);
    // Refetch backend user to clear workshop context
    retryBackendFetch();
  }, [isInWorkshopMode, setActiveWorkshop, retryBackendFetch]);

  const value: WorkshopModeContextType = {
    isInWorkshopMode,
    activeWorkshopId,
    activeWorkshopName,
    enterWorkshopMode,
    exitWorkshopMode,
    isLoading: setActiveWorkshop.isPending,
  };

  return (
    <WorkshopModeContext.Provider value={value}>
      {children}
    </WorkshopModeContext.Provider>
  );
}
