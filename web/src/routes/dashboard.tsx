import { createFileRoute } from '@tanstack/react-router';
import { DashboardContent } from '@/features/dashboard/components/Dashboard';
import { ROUTES } from '@/common/routes/routes';

export const Route = createFileRoute(ROUTES.DASHBOARD)({
  component: DashboardPage,
});

function DashboardPage() {
  return <DashboardContent />;
}
