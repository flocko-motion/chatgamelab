import { createFileRoute } from '@tanstack/react-router';
import { ServerSettings } from '@/features/admin/components/ServerSettings';

export const Route = createFileRoute('/admin/server-settings/')({
  component: ServerSettingsPage,
});

function ServerSettingsPage() {
  return <ServerSettings />;
}
