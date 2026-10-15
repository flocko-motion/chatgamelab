import { useState } from "react";
import {
  ActionIcon,
  Button,
  Card,
  Group,
  Stack,
  Text,
  TextInput,
  Tooltip,
} from "@mantine/core";
import {
  IconArrowDown,
  IconArrowUp,
  IconDeviceFloppy,
  IconPlus,
  IconTrash,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { ObjPublicWorkshopLink } from "@/api/generated";
import { PUBLIC_LINKS_MAX } from "@/common/lib/publicWorkshop";

interface PublicPageLinksEditorProps {
  links: ObjPublicWorkshopLink[];
  onSave: (links: ObjPublicWorkshopLink[]) => Promise<void>;
  saving: boolean;
}

const EMPTY: ObjPublicWorkshopLink = { title: "", description: "", url: "" };

/** Further reading shown below the games on the public page. */
export function PublicPageLinksEditor({
  links,
  onSave,
  saving,
}: PublicPageLinksEditorProps) {
  const { t } = useTranslation("common");
  const [rows, setRows] = useState<ObjPublicWorkshopLink[]>(links);
  const [error, setError] = useState<string | null>(null);

  const edit = (index: number, patch: Partial<ObjPublicWorkshopLink>) =>
    setRows(rows.map((row, i) => (i === index ? { ...row, ...patch } : row)));

  const move = (index: number, to: number) => {
    const next = [...rows];
    [next[index], next[to]] = [next[to], next[index]];
    setRows(next);
  };

  const save = async () => {
    try {
      await onSave(rows);
      setError(null);
    } catch {
      setError(t("myOrganization.publicPage.links.error"));
    }
  };

  const dirty = JSON.stringify(rows) !== JSON.stringify(links);

  return (
    <Stack gap="xs">
      <div>
        <Text size="sm" fw={500}>
          {t("myOrganization.publicPage.links.title")}
        </Text>
        <Text size="xs" c="dimmed">
          {t("myOrganization.publicPage.links.hint")}
        </Text>
      </div>

      {rows.map((row, index) => (
        <Card key={index} padding="xs" radius="sm" withBorder>
          <Stack gap="xs">
            <Group gap="xs" wrap="nowrap" align="flex-end">
              <TextInput
                size="xs"
                style={{ flex: 1 }}
                label={t("myOrganization.publicPage.links.linkTitle")}
                value={row.title ?? ""}
                onChange={(e) => edit(index, { title: e.currentTarget.value })}
              />
              <Tooltip label={t("myOrganization.publicPage.links.up")}>
                <ActionIcon
                  variant="subtle"
                  size="sm"
                  disabled={index === 0}
                  onClick={() => move(index, index - 1)}
                  aria-label={t("myOrganization.publicPage.links.up")}
                >
                  <IconArrowUp size={14} />
                </ActionIcon>
              </Tooltip>
              <Tooltip label={t("myOrganization.publicPage.links.down")}>
                <ActionIcon
                  variant="subtle"
                  size="sm"
                  disabled={index === rows.length - 1}
                  onClick={() => move(index, index + 1)}
                  aria-label={t("myOrganization.publicPage.links.down")}
                >
                  <IconArrowDown size={14} />
                </ActionIcon>
              </Tooltip>
              <Tooltip label={t("myOrganization.publicPage.links.remove")}>
                <ActionIcon
                  variant="subtle"
                  color="red"
                  size="sm"
                  onClick={() => setRows(rows.filter((_, i) => i !== index))}
                  aria-label={t("myOrganization.publicPage.links.remove")}
                >
                  <IconTrash size={14} />
                </ActionIcon>
              </Tooltip>
            </Group>
            <TextInput
              size="xs"
              label={t("myOrganization.publicPage.links.url")}
              placeholder="https://"
              value={row.url ?? ""}
              onChange={(e) => edit(index, { url: e.currentTarget.value })}
            />
            <TextInput
              size="xs"
              label={t("myOrganization.publicPage.links.linkDescription")}
              value={row.description ?? ""}
              onChange={(e) => edit(index, { description: e.currentTarget.value })}
            />
          </Stack>
        </Card>
      ))}

      {error && (
        <Text size="xs" c="red">
          {error}
        </Text>
      )}

      <Group gap="xs">
        <Button
          size="xs"
          variant="light"
          leftSection={<IconPlus size={14} />}
          disabled={rows.length >= PUBLIC_LINKS_MAX}
          onClick={() => setRows([...rows, { ...EMPTY }])}
        >
          {t("myOrganization.publicPage.links.add")}
        </Button>
        <Button
          size="xs"
          leftSection={<IconDeviceFloppy size={14} />}
          disabled={!dirty}
          loading={saving}
          onClick={save}
        >
          {t("myOrganization.publicPage.links.save")}
        </Button>
      </Group>
    </Stack>
  );
}
