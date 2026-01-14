import { createFileRoute } from '@tanstack/react-router';
import { DashboardContent } from '@/features/dashboard/components/Dashboard';

export const Route = createFileRoute('/dashboard')({
  component: DashboardPage,
});

function DashboardPage() {
  return <DashboardContent />;
}
