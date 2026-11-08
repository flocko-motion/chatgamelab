import { createFileRoute, useSearch } from '@tanstack/react-router';
import { MyOrganization } from '@/features/my-organization';

type OrgSearch = {
  action?: 'create-workshop';
};

export const Route = createFileRoute('/my-organization/')({
  component: MyOrganizationPage,
  validateSearch: (search: Record<string, unknown>): OrgSearch => ({
    action: search.action === 'create-workshop' ? 'create-workshop' : undefined,
  }),
});

function MyOrganizationPage() {
  const { action } = useSearch({ from: '/my-organization/' });
  return <MyOrganization autoCreateWorkshop={action === 'create-workshop'} />;
}
