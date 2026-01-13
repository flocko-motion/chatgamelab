import { createFileRoute } from '@tanstack/react-router';
import { Dashboard } from '@/features/dashboard/components/Dashboard';

export const Route = createFileRoute('/dashboard')({
  component: DashboardPage,
});

function DashboardPage() {
  return <Dashboard />;
}
