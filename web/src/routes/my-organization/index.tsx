import { createFileRoute } from '@tanstack/react-router';
import { MyOrganization } from '@/features/my-organization';

export const Route = createFileRoute('/my-organization/')({
  component: MyOrganizationPage,
});

function MyOrganizationPage() {
  return <MyOrganization />;
}
