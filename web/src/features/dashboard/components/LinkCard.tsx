import { Card, Group, Text, ThemeIcon, Stack, UnstyledButton } from '@mantine/core';
import { IconExternalLink } from '@tabler/icons-react';
import type { ReactNode } from 'react';
import { CardTitle } from '@components/typography';
import classes from './LinkCard.module.css';

interface LinkCardProps {
  title: string;
  description?: string;
  href: string;
  icon?: ReactNode;
}

export function LinkCard({ title, description, href, icon }: LinkCardProps) {
  return (
    <Card
      component="a"
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      p="lg"
      withBorder
      shadow="sm"
      className={classes.card}
    >
      <Group justify="space-between" align="flex-start" wrap="nowrap">
        <Group gap="sm" wrap="nowrap" style={{ flex: 1, minWidth: 0 }}>
          {icon && (
            <ThemeIcon color="accent" size={32} radius="sm" variant="light">
              {icon}
            </ThemeIcon>
          )}
          <div style={{ minWidth: 0 }}>
            <CardTitle>{title}</CardTitle>
            {description && (
              <Text size="sm" c="gray.5" mt={4} lineClamp={2}>
                {description}
              </Text>
            )}
          </div>
        </Group>
        <ThemeIcon color="gray" size={24} variant="subtle">
          <IconExternalLink size={16} />
        </ThemeIcon>
      </Group>
    </Card>
  );
}

export interface LinkItem {
  id: string;
  title: string;
  description?: string;
  href: string;
  icon?: ReactNode;
}

interface LinksCardProps {
  title: string;
  links: LinkItem[];
}

export function LinksCard({ title, links }: LinksCardProps) {
  return (
    <Card p="lg" withBorder shadow="sm" h="100%">
      <CardTitle>{title}</CardTitle>
      <Stack gap="xs" mt="md">
        {links.map((link) => (
          <UnstyledButton
            key={link.id}
            component="a"
            href={link.href}
            target="_blank"
            rel="noopener noreferrer"
            className={classes.linkItem}
          >
            <Group justify="space-between" align="center" wrap="nowrap">
              <Group gap="sm" wrap="nowrap" style={{ flex: 1, minWidth: 0 }}>
                {link.icon && (
                  <ThemeIcon color="accent" size={28} radius="sm" variant="light">
                    {link.icon}
                  </ThemeIcon>
                )}
                <div style={{ minWidth: 0 }}>
                  <Text size="sm" fw={500}>
                    {link.title}
                  </Text>
                  {link.description && (
                    <Text size="xs" c="gray.5">
                      {link.description}
                    </Text>
                  )}
                </div>
              </Group>
              <IconExternalLink size={14} color="var(--mantine-color-gray-5)" />
            </Group>
          </UnstyledButton>
        ))}
      </Stack>
    </Card>
  );
}
