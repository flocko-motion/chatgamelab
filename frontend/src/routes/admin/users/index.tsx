import { createFileRoute } from '@tanstack/react-router';
import { UserManagement } from '@/features/admin/components/UserManagement';

export const Route = createFileRoute('/admin/users/')({
  component: UsersPage,
});

function UsersPage() {
  return <UserManagement />;
}
