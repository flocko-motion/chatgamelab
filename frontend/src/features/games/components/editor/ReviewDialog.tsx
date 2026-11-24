import { Group, Modal, Stack, ThemeIcon } from "@mantine/core";
import { IconMessageCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, TextButton } from "@components/buttons";
import { SectionTitle } from "@components/typography";
import type { EditorPage } from "../../lib/editorPages";
import type { ReviewNote } from "../../hooks/useGameReview";

/**
 * The coach's answer: one point, big enough to read, then Later or Go to
 * the section; with nothing to fix, Looks good and Close.
 */
export function ReviewDialog({
  note,
  onClose,
  onGo,
}: {
  note: ReviewNote | null;
  onClose: () => void;
  onGo: (page: EditorPage) => void;
}) {
  const { t } = useTranslation("common");
  const page = note?.page ?? null;
  let color = "green";
  if (page) color = "yellow";
  else if (note?.failed) color = "gray";
  return (
    <Modal opened={!!note} onClose={onClose} centered size="md" withCloseButton={false} radius="lg" zIndex={400}>
      {note && (
        <Stack align="center" gap="lg" p="md" ta="center">
          <ThemeIcon size={72} radius="xl" variant="light" color={color}>
            <IconMessageCircle size={40} />
          </ThemeIcon>
          <SectionTitle>{note.text}</SectionTitle>
          <Group justify="center" gap="sm">
            <TextButton onClick={onClose}>{page ? t("games.editor.review.later") : t("close")}</TextButton>
            {page && (
              <ActionButton onClick={() => onGo(page)}>
                {t("games.editor.review.goTo", { page: t(`games.editor.pages.${page}.label`) })}
              </ActionButton>
            )}
          </Group>
        </Stack>
      )}
    </Modal>
  );
}
