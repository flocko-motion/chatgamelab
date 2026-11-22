import { Group } from "@mantine/core";
import { IconCircleCheck, IconCircleMinus, IconCircleX } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { HelperText } from "@components/typography";

/** Whether a key worked when it was last used: working, failed, or not used yet. */
export function KeyStatus({ success }: { success: boolean | undefined }) {
  const { t } = useTranslation("common");
  const [icon, color, label] =
    success === true
      ? [<IconCircleCheck size={14} />, "green.7", t("apiKeys.status.working")]
      : success === false
        ? [<IconCircleX size={14} />, "red.7", t("apiKeys.status.failed")]
        : [<IconCircleMinus size={14} />, "gray.6", t("apiKeys.status.unknown")];
  return (
    <Group gap={4} wrap="nowrap" c={color}>
      {icon}
      <HelperText c={color}>{label}</HelperText>
    </Group>
  );
}
