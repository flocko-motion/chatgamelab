import { Alert, Box } from "@mantine/core";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/providers/AuthProvider";

interface TierCappedNoticeProps {
  /** The tier the session plays at */
  applied: string | null;
  /** The tier it asked for, set only while a free-use key caps it lower */
  requested: string | null;
  /** Who released that key: "organization" or "server" */
  limitedBy: string | null;
}

/**
 * Says that the game plays at a lower AI quality tier than chosen, because the
 * free-use key paying for it is released only up to that tier, and what to do.
 * Renders nothing while the session plays at the tier it asked for.
 */
export function TierCappedNotice({
  applied,
  requested,
  limitedBy,
}: TierCappedNoticeProps) {
  const { t } = useTranslation("common");
  const { backendUser } = useAuth();
  if (!applied || !requested) return null;

  const source = limitedBy === "server" ? "server" : "organization";
  // Participants cannot change keys or tiers; their workshop leaders can.
  const audience =
    backendUser?.role?.role === "participant" ? "participant" : source;
  const tierLabel = (tier: string) => t(`aiQualityTier.${tier}`);

  return (
    <Box px="md" pt="xs">
      <Alert
        color="yellow"
        variant="light"
        icon={<IconAlertCircle size={16} />}
        title={t("gamePlayer.tierCapped.title", {
          applied: tierLabel(applied),
          requested: tierLabel(requested),
        })}
      >
        {t(`gamePlayer.tierCapped.reason.${source}`, {
          tier: tierLabel(applied),
        })}{" "}
        {t(`gamePlayer.tierCapped.action.${audience}`)}
      </Alert>
    </Box>
  );
}
