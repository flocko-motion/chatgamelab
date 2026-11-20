import { createFileRoute } from '@tanstack/react-router';
import { ApiKeyManagement } from '@/features/api-keys/components/ApiKeyManagement';
import { ROUTES } from '@/common/routes/routes';

export const Route = createFileRoute(ROUTES.API_KEYS)({
  component: ApiKeysPage,
});

function ApiKeysPage() {
  return <ApiKeyManagement />;
}
