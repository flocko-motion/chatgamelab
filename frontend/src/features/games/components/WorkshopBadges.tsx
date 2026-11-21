import { Badge, Group, Skeleton } from "@mantine/core";
import { useWorkshop } from "@/api/hooks";

/** The workshop a game belongs to, and its organisation, as badges. */
export function WorkshopBadges({ workshopId }: { workshopId: string }) {
  const { data: workshop, isLoading } = useWorkshop(workshopId);
  if (isLoading) return <Skeleton height={14} width={120} radius="xl" />;
  if (!workshop) return null;

  return (
    <Group gap={4} wrap="wrap">
      <Badge size="xs" color="violet" variant="light" maw={160}>
        {workshop.name}
      </Badge>
      {workshop.institution?.name && (
        <Badge size="xs" color="gray" variant="light" maw={160}>
          {workshop.institution.name}
        </Badge>
      )}
    </Group>
  );
}
