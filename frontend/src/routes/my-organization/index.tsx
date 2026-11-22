import { createFileRoute, useSearch } from '@tanstack/react-router';
import { MyOrganization, ORGANISATION_TABS, type OrganisationTab } from '@/features/my-organization';
import { validateListSearch, type ListSearch } from '@/common/lib/listQuery';

type OrgSearch = ListSearch & {
  action?: 'create-workshop';
  tab?: OrganisationTab;
};

export const Route = createFileRoute('/my-organization/')({
  component: MyOrganizationPage,
  // The open tab's list keeps its query in the URL.
  validateSearch: (search: Record<string, unknown>): OrgSearch => ({
    ...validateListSearch(search),
    action: search.action === 'create-workshop' ? 'create-workshop' : undefined,
    tab: ORGANISATION_TABS.find((t) => t === search.tab),
  }),
});

function MyOrganizationPage() {
  const { action, tab } = useSearch({ from: '/my-organization/' });
  return <MyOrganization autoCreateWorkshop={action === 'create-workshop'} tab={tab} />;
}
