import { createFileRoute } from '@tanstack/react-router';
import { MyWorkshop } from '@/features/my-workshop';
import { validateListSearch } from '@/common/lib/listQuery';

export const Route = createFileRoute('/my-workshop/')({
  component: MyWorkshopPage,
  validateSearch: validateListSearch,
});

function MyWorkshopPage() {
  return <MyWorkshop />;
}
