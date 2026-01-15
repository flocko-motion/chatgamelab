import { Table, Card, Stack, Group, Text, Box, Skeleton } from '@mantine/core';
import { useMediaQuery } from '@mantine/hooks';
import type { ReactNode } from 'react';

export interface DataTableColumn<T> {
  key: string;
  header: string;
  width?: number | string;
  render: (item: T) => ReactNode;
  hideOnMobile?: boolean;
}

export interface DataTableProps<T> {
  data: T[];
  columns: DataTableColumn<T>[];
  getRowKey: (item: T) => string;
  onRowClick?: (item: T) => void;
  renderMobileCard: (item: T) => ReactNode;
  emptyState?: ReactNode;
  isLoading?: boolean;
  loadingRows?: number;
  maxHeight?: number | string;
  fillHeight?: boolean;
}

const tableHeaderStyle = {
  color: 'var(--mantine-color-gray-6)',
  fontWeight: 600,
  fontSize: '0.75rem',
  textTransform: 'uppercase' as const,
  letterSpacing: '0.5px',
};

export function DataTable<T>({
  data,
  columns,
  getRowKey,
  onRowClick,
  renderMobileCard,
  emptyState,
  isLoading = false,
  loadingRows = 3,
  maxHeight,
  fillHeight = false,
}: DataTableProps<T>) {
  const isMobile = useMediaQuery('(max-width: 48em)');

  if (isLoading) {
    if (isMobile) {
      return (
        <Stack gap="md">
          {Array.from({ length: loadingRows }).map((_, i) => (
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
      );
    }
    return <Skeleton height={300} />;
  }

  if (data.length === 0 && emptyState) {
    return <>{emptyState}</>;
  }

  if (isMobile) {
    return (
      <Stack gap="md">
        {data.map((item) => (
          <Box key={getRowKey(item)}>{renderMobileCard(item)}</Box>
        ))}
      </Stack>
    );
  }

  const effectiveMaxHeight = fillHeight ? '100%' : maxHeight;
  const useScrolling = fillHeight || !!maxHeight;

  return (
    <Card shadow="sm" p={0} radius="md" withBorder style={fillHeight ? { flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' } : undefined}>
      <Table.ScrollContainer minWidth={500} mah={effectiveMaxHeight} style={useScrolling ? { overflowY: 'auto', flex: 1 } : undefined}>
        <Table verticalSpacing="md" horizontalSpacing="lg" stickyHeader={useScrolling}>
          <Table.Thead>
            <Table.Tr style={{ borderBottom: '2px solid var(--mantine-color-gray-2)' }}>
              {columns
                .filter((col) => !col.hideOnMobile || !isMobile)
                .map((column) => (
                  <Table.Th key={column.key} w={column.width} style={tableHeaderStyle}>
                    {column.header}
                  </Table.Th>
                ))}
            </Table.Tr>
          </Table.Thead>
          <Table.Tbody>
            {data.map((item) => (
              <Table.Tr
                key={getRowKey(item)}
                style={{
                  cursor: onRowClick ? 'pointer' : 'default',
                  transition: 'background-color 150ms ease',
                }}
                onClick={() => onRowClick?.(item)}
              >
                {columns
                  .filter((col) => !col.hideOnMobile || !isMobile)
                  .map((column) => (
                    <Table.Td key={column.key}>{column.render(item)}</Table.Td>
                  ))}
              </Table.Tr>
            ))}
          </Table.Tbody>
        </Table>
      </Table.ScrollContainer>
    </Card>
  );
}

export function DataTableEmptyState({
  icon,
  title,
  description,
  action,
}: {
  icon: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <Card shadow="sm" p="xl" radius="md" withBorder>
      <Stack align="center" gap="md" py="xl">
        {icon}
        <Text c="gray.6" ta="center">
          {title}
        </Text>
        {description && (
          <Text size="sm" c="gray.5" ta="center">
            {description}
          </Text>
        )}
        {action}
      </Stack>
    </Card>
  );
}
