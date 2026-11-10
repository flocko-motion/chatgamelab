import { Alert, Box, Text } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { TextButton } from "./buttons/TextButton";

interface AiTierNoticeProps {
  /** The selected tier that will not take effect */
  tier: string;
  /** Why the tier does not take effect */
  reason: string;
  /** Optional label for a button that leads to the fix */
  actionLabel?: string;
  onAction?: () => void;
}

/**
 * Warns that a selected AI quality tier has no effect, because a tier only
 * applies together with the API key configured at the same place.
 *
 * @example
 * <AiTierNotice tier="max" reason={t("aiQualityTier.notUsed.noWorkshopKey")} />
 */
export function AiTierNotice({
  tier,
  reason,
  actionLabel,
  onAction,
}: AiTierNoticeProps) {
  const { t } = useTranslation("common");

  return (
    <Alert
      color="yellow"
      variant="light"
      icon={<IconAlertCircle size={16} />}
      title={t("aiQualityTier.notUsed.title")}
    >
      <Text size="sm">{reason}</Text>
      {tier === "max" && (
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
