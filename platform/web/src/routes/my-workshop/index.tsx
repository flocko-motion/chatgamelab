import { createFileRoute } from '@tanstack/react-router';
import { MyWorkshop } from '@/features/my-workshop';

export const Route = createFileRoute('/my-workshop/')({
  component: MyWorkshopPage,
});

function MyWorkshopPage() {
  return <MyWorkshop />;
}
