import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';
import { DashboardContent } from '@/features/dashboard/components/Dashboard';
import { ROUTES } from '@/common/routes/routes';
import { useWorkshopMode } from '@/providers/WorkshopModeProvider';
import { useAuth } from '@/providers/AuthProvider';

export const Route = createFileRoute(ROUTES.DASHBOARD)({
  component: DashboardPage,
});

function DashboardPage() {
  const navigate = useNavigate();
  const { isInWorkshopMode } = useWorkshopMode();
  const { isParticipant, isLoading } = useAuth();

  // Redirect workshop mode users to /my-workshop
  useEffect(() => {
    if (!isLoading && (isInWorkshopMode || isParticipant)) {
      navigate({ to: ROUTES.MY_WORKSHOP as '/' });
    }
  }, [isLoading, isInWorkshopMode, isParticipant, navigate]);

  // Show nothing while redirecting
  if (!isLoading && (isInWorkshopMode || isParticipant)) {
    return null;
  }

  return <DashboardContent />;
}
