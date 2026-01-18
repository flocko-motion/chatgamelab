import { Card, Stack, Group, Skeleton, Text, UnstyledButton, Box } from '@mantine/core';
import { IconChevronRight } from '@tabler/icons-react';
import { CardTitle } from '@components/typography';
import { TextButton } from '@components/buttons';
import classes from './InformationalCard.module.css';

export interface ListItem {
  id: string;
  label: string;
  sublabel?: string;
  onClick?: () => void;
}

interface InformationalCardProps {
  title: string;
  items: ListItem[];
  emptyMessage: string;
  viewAllLabel?: string;
  onViewAll?: () => void;
  isLoading?: boolean;
  maxItems?: number;
}

export function InformationalCard({
  title,
  items,
  emptyMessage,
  viewAllLabel,
  onViewAll,
  isLoading = false,
  maxItems = 5,
}: InformationalCardProps) {
  const displayItems = items.slice(0, maxItems);

  return (
    <Card p="lg" withBorder shadow="sm" h="100%">
      <Group justify="space-between" mb="md" wrap="nowrap" align="center" gap="md">
        <Box style={{ flex: '0 1 auto', minWidth: 0 }}>
          <CardTitle>{title}</CardTitle>
        </Box>
        {viewAllLabel && onViewAll && (
          <Box style={{ flexShrink: 0 }}>
            <TextButton onClick={onViewAll} rightSection={<IconChevronRight size={14} />}>
              {viewAllLabel}
            </TextButton>
          </Box>
        )}
      </Group>

      <Stack gap="xs">
        {isLoading ? (
          <>
            {[1, 2, 3].map((i) => (
              <Skeleton key={i} height={40} radius="sm" />
            ))}
          </>
        ) : displayItems.length === 0 ? (
          <Text c="gray.5" size="sm" ta="center" py="md">
            {emptyMessage}
          </Text>
        ) : (
          displayItems.map((item) => (
            <UnstyledButton
              key={item.id}
              onClick={item.onClick}
              className={classes.listItem}
              data-clickable={!!item.onClick}
            >
              <Group justify="space-between" wrap="nowrap" gap="xs">
                <Text size="sm" fw={500} lineClamp={1} style={{ flex: 1, minWidth: 0 }}>
                  {item.label}
                </Text>
                {item.sublabel && (
                  <Text size="xs" c="gray.5" style={{ flexShrink: 0 }}>
                    {item.sublabel}
                  </Text>
                )}
              </Group>
            </UnstyledButton>
          ))
        )}
      </Stack>
    </Card>
  );
}
