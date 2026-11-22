import { createFileRoute } from '@tanstack/react-router';
import { ApiKeyManagement } from '@/features/api-keys/components/ApiKeyManagement';
import { ROUTES } from '@/common/routes/routes';
import { validateListSearch } from '@/common/lib/listQuery';

export const Route = createFileRoute(ROUTES.API_KEYS)({
  validateSearch: validateListSearch,
  component: ApiKeysPage,
});

function ApiKeysPage() {
  return <ApiKeyManagement />;
}
