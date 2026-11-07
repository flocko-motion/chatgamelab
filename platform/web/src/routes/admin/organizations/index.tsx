import { createFileRoute } from '@tanstack/react-router';
import { OrganizationsManagement } from '@/features/admin/components/OrganizationsManagement';

export const Route = createFileRoute('/admin/organizations/')({
  component: OrganizationsPage,
});

function OrganizationsPage() {
  return <OrganizationsManagement />;
}
