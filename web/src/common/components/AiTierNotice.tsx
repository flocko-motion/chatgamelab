import { Alert, Box, Text } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { TextButton } from "./buttons/TextButton";
import { tiersAboveLimit } from "@/common/lib/aiQualityTier";

interface AiTierNoticeProps {
  title: string;
  /** Why, and what to do about it */
  reason: string;
  /** Adds that narration (audio) needs 'Max' */
  narrationHint?: boolean;
  /** Optional label for a button that leads to the fix */
  actionLabel?: string;
  onAction?: () => void;
}

/**
 * Yellow notice about AI quality tiers: one that has no effect, or ones a
 * free-use key is not released for.
 *
 * @example
 * <AiTierNotice title={t("aiQualityTier.notUsed.title")} reason={t("aiQualityTier.notUsed.noServerFreeKey")} />
 */
export function AiTierNotice({
  title,
  reason,
  narrationHint,
  actionLabel,
  onAction,
}: AiTierNoticeProps) {
  const { t } = useTranslation("common");

  return (
    <Alert
      color="yellow"
      variant="light"
      icon={<IconAlertCircle size={16} />}
      title={title}
    >
      <Text size="sm">{reason}</Text>
      {narrationHint && (
        <Text size="sm" mt={4}>
          {t("aiQualityTier.notUsed.narrationOnlyMax")}
        </Text>
      )}
      {actionLabel && onAction && (
        <Box mt="xs" ml={-8}>
          <TextButton size="xs" onClick={onAction}>
            {actionLabel}
          </TextButton>
        </Box>
      )}
    </Alert>
  );
}

interface AiTierLimitNoticeProps {
  /** The highest tier the free-use key is released up to */
  limit: string;
  /** Who released it */
  source: "organization" | "server";
  /** Whose tier is limited: a workshop's, or the user's own */
  audience: "workshop" | "user";
  actionLabel?: string;
  onAction?: () => void;
}

/**
 * Names the tiers a free-use key is not released for, why, and how to get them.
 * Renders nothing while every tier is released.
 */
export function AiTierLimitNotice({
  limit,
  source,
  audience,
  actionLabel,
  onAction,
}: AiTierLimitNoticeProps) {
  const { t } = useTranslation("common");
  const above = tiersAboveLimit(limit);
  if (above.length === 0) return null;

  return (
    <AiTierNotice
      title={t("aiQualityTier.limit.title", {
        tiers: above.map((tier) => t(`aiQualityTier.${tier}`)).join(", "),
      })}
      reason={t(`aiQualityTier.limit.${audience}.${source}`, {
        tier: t(`aiQualityTier.${limit}`),
      })}
      narrationHint={above.includes("max")}
      actionLabel={actionLabel}
      onAction={onAction}
    />
  );
}
