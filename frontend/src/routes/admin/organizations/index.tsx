import { createFileRoute } from '@tanstack/react-router';
import { validateListSearch } from '@/common/lib/listQuery';
import { OrganizationsManagement } from '@/features/admin/components/OrganizationsManagement';

export const Route = createFileRoute('/admin/organizations/')({
  validateSearch: validateListSearch,
  component: OrganizationsPage,
});

function OrganizationsPage() {
  return <OrganizationsManagement />;
}
