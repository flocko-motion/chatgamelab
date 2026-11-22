import { createFileRoute } from '@tanstack/react-router';
import { validateListSearch } from '@/common/lib/listQuery';
import { UserManagement } from '@/features/admin/components/UserManagement';

export const Route = createFileRoute('/admin/users/')({
  validateSearch: validateListSearch,
  component: UsersPage,
});

function UsersPage() {
  return <UserManagement />;
}
