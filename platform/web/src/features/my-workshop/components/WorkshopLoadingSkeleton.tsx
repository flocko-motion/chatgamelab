import { Stack, Card, Group, Skeleton } from "@mantine/core";

interface WorkshopLoadingSkeletonProps {
  isMobile: boolean;
}

export function WorkshopLoadingSkeleton({ isMobile }: WorkshopLoadingSkeletonProps) {
  return (
    <Stack gap="xl">
      <Skeleton height={40} width="50%" />
      <Skeleton height={36} width={180} />
      {isMobile ? (
        <Stack gap="md">
          {[1, 2, 3].map((i) => (
            <Card key={i} shadow="sm" p="lg" radius="md" withBorder>
              <Stack gap="sm">
                <Skeleton height={24} width="70%" />
                <Skeleton height={16} width="90%" />
                <Group gap="xl">
                  <Skeleton height={32} width={80} />
                  <Skeleton height={32} width={80} />
                </Group>
              </Stack>
            </Card>
          ))}
        </Stack>
      ) : (
        <Skeleton height={300} />
      )}
    </Stack>
  );
}
