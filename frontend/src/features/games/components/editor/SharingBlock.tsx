import type { ReactNode } from "react";
import { Group, Stack } from "@mantine/core";
import { HelperText, ItemTitle } from "@components/typography";

interface SharingBlockProps {
  title: string;
  description: string;
  /** What switches or runs the block (a switch, a button), beside its title. */
  control?: ReactNode;
  children?: ReactNode;
}

/**
 * One block of the Sharing page: a title, what it does, its control beside
 * them, and below what it holds. Every block heads the same way, so the page
 * reads as a list of equal choices.
 */
export function SharingBlock({ title, description, control, children }: SharingBlockProps) {
  return (
    <Stack gap="sm">
      <Group justify="space-between" wrap="nowrap" align="flex-start" gap="md">
        <Stack gap={2} style={{ flex: 1, minWidth: 0 }}>
          <ItemTitle>{title}</ItemTitle>
          <HelperText>{description}</HelperText>
        </Stack>
        {control && <div style={{ flexShrink: 0 }}>{control}</div>}
      </Group>
      {children}
    </Stack>
  );
}
