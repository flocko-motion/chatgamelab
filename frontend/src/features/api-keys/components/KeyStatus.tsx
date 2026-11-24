import { Group, Loader } from "@mantine/core";
import { IconCircleCheck, IconCircleMinus, IconCircleX } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { HelperText } from "@components/typography";
import classes from "./KeyStatus.module.css";

export interface KeyStatusProps {
  /** Whether the key worked when it was last used or checked; undefined: neither yet. */
  success: boolean | undefined;
  /** A check is running. */
  checking?: boolean;
  /**
   * When the last check here finished: a new value flashes the status, so a
   * check that found the same status is seen to have run.
   */
  checkedAt?: number;
}

/** Whether a key works: working, failed, or not used yet; or being checked. */
export function KeyStatus({ success, checking = false, checkedAt }: KeyStatusProps) {
  const { t } = useTranslation("common");
  if (checking) {
    return (
      <Group gap={4} wrap="nowrap" c="gray.6" aria-live="polite">
        <Loader size={12} color="gray" />
        <HelperText c="gray.6">{t("apiKeys.status.checking")}</HelperText>
      </Group>
    );
  }
  const [icon, color, label] =
    success === true
      ? [<IconCircleCheck size={14} />, "green.7", t("apiKeys.status.working")]
      : success === false
        ? [<IconCircleX size={14} />, "red.7", t("apiKeys.status.failed")]
        : [<IconCircleMinus size={14} />, "gray.6", t("apiKeys.status.unknown")];
  return (
    <Group
      // A new key per check restarts the flash.
      key={checkedAt ?? "unchecked"}
      className={checkedAt ? classes.flash : undefined}
      gap={4}
      wrap="nowrap"
      c={color}
      aria-live="polite"
    >
      {icon}
      <HelperText c={color}>{label}</HelperText>
    </Group>
  );
}
