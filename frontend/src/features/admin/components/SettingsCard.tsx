import { Card, Group, Loader, Stack } from "@mantine/core";
import type { ReactNode } from "react";
import { CardTitle, HelperText } from "@components/typography";

/** One group of site settings: an icon, a title, what it is for, and its fields. */
export function SettingsCard({
  icon,
  title,
  description,
  busy,
  children,
}: {
  icon: ReactNode;
  title: string;
  description?: string;
  busy?: boolean;
  children: ReactNode;
}) {
  return (
    <Card shadow="sm" padding="lg" radius="md" withBorder>
      <Stack gap="md">
        <Stack gap={4}>
          <Group gap="xs">
            {icon}
            <CardTitle>{title}</CardTitle>
            {busy && <Loader size="xs" />}
          </Group>
          {description && <HelperText>{description}</HelperText>}
        </Stack>
        {children}
      </Stack>
    </Card>
  );
}
