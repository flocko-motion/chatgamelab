import { Group } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { HelperText } from "@components/typography";
import type { Problem } from "@/common/lib/playability";

/** Says, beside a disabled Play button, why the game cannot be played. */
export function UnplayableNote({ problem }: { problem: Problem }) {
  const { t } = useTranslation("common");
  return (
    <Group gap={4} wrap="nowrap" align="flex-start">
      <IconAlertCircle
        size={14}
        color="var(--mantine-color-orange-7)"
        aria-hidden
        style={{ flexShrink: 0, marginTop: 3 }}
      />
      <HelperText c="orange.8">{t(`play.unplayable.${problem}`)}</HelperText>
    </Group>
  );
}
