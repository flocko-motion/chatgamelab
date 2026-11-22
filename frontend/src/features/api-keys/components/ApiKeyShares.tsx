import { Stack, Group, UnstyledButton, Badge } from "@mantine/core";
import { IconBuilding, IconHeartFilled, IconExternalLink, IconLink } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import { DeleteIconButton } from "@components/buttons";
import { BodyText, HelperText, Label } from "@components/typography";
import { useRemoveGameSponsor, useRevokePrivateShare } from "@/api/hooks";
import type { KeyShareView } from "@/api/types";
import { keyShareGroups, type ShareLinkEntry } from "../lib/keyShareGroups";

interface ApiKeySharesProps {
  shares: KeyShareView[];
}

/** What a key is shared as, by kind: organisations, sponsored games, share links. */
export function ApiKeyShares({ shares }: ApiKeySharesProps) {
  const { organisations, sponsorships, shareLinks } = keyShareGroups(shares);

  return (
    <Stack gap="sm">
      {organisations.length > 0 && <OrgSharesSection shares={organisations} />}
      {sponsorships.length > 0 && <SponsorshipsSection shares={sponsorships} />}
      {shareLinks.length > 0 && <GameSharesSection shares={shareLinks} />}
    </Stack>
  );
}

function OrgSharesSection({ shares }: { shares: KeyShareView[] }) {
  const { t } = useTranslation("common");
  const navigate = useNavigate();

  return (
    <Stack gap={4}>
      <Group gap={8} align="center">
        <IconBuilding size={18} />
        <Label uppercase>{t("apiKeys.shares.orgShares")} ({shares.length})</Label>
      </Group>
      <Stack gap={6} pl="md">
        {shares.map((share) => {
          const name = share.institution?.name || "-";
          return (
            <Group key={share.id} gap="sm" align="center">
              <BodyText size="sm" fw={500}>{name}</BodyText>
              {share.institution?.id && (
                <UnstyledButton
                  onClick={() => navigate({ to: "/my-organization" })}
                >
                  <Group gap={4} align="center">
                    <IconExternalLink
                      size={16}
                      color="var(--mantine-color-accent-5)"
                    />
                    <BodyText size="sm" c="accent.5" fw={500}>{t("apiKeys.shares.goToOrg")}</BodyText>
                  </Group>
                </UnstyledButton>
              )}
            </Group>
          );
        })}
      </Stack>
    </Stack>
  );
}

function SponsorshipsSection({ shares }: { shares: KeyShareView[] }) {
  const { t } = useTranslation("common");
  const removeGameSponsor = useRemoveGameSponsor();

  return (
    <Stack gap={4}>
      <Group gap={8} align="center">
        <IconHeartFilled size={18} color="var(--mantine-color-pink-5)" />
        <Label uppercase>{t("apiKeys.shares.sponsorships")} ({shares.length})</Label>
      </Group>
      <Stack gap={6} pl="md">
        {shares.map((share) => (
          <Group key={share.id} gap="sm" align="center" justify="space-between">
            <BodyText size="sm">{share.game?.name || t("apiKeys.shares.unknownGame")}</BodyText>
            {share.mayRevoke && (
              <DeleteIconButton
                size="sm"
                onClick={() => {
                  if (share.game?.id) {
                    removeGameSponsor.mutate(share.game.id);
                  }
                }}
                aria-label={t("apiKeys.shares.removeSponsor")}
              />
            )}
          </Group>
        ))}
      </Stack>
    </Stack>
  );
}

function GameSharesSection({ shares }: { shares: ShareLinkEntry[] }) {
  const { t } = useTranslation("common");
  const revokeShare = useRevokePrivateShare();

  const getRemainingLabel = (share: KeyShareView): string => {
    if (share.remaining == null) return t("apiKeys.shares.unlimited");
    return t("apiKeys.shares.remaining", { count: share.remaining });
  };

  return (
    <Stack gap={4}>
      <Group gap={8} align="center">
        <IconLink size={18} color="var(--mantine-color-accent-5)" />
        <Label uppercase>{t("apiKeys.shares.gameShares")} ({shares.length})</Label>
      </Group>
      <Stack gap={6} pl="md">
        {shares.map(({ share, source }) => (
          <Group
            key={share.gameShareId || share.id}
            gap="sm"
            align="center"
            justify="space-between"
          >
            <Group gap="xs">
              <BodyText size="sm" fw={500}>{share.game?.name || t("apiKeys.shares.unknownGame")}</BodyText>
              <Badge size="xs" variant="light" color="gray">
                {source === "organization"
                  ? t("apiKeys.shares.context.organization")
                  : t("apiKeys.shares.context.personal")}
              </Badge>
              <HelperText>{getRemainingLabel(share)}</HelperText>
            </Group>
            {share.mayRevoke && (
              <DeleteIconButton
                size="sm"
                onClick={() => {
                  if (share.game?.id && share.gameShareId) {
                    revokeShare.mutate({
                      gameId: share.game.id,
                      shareId: share.gameShareId,
                    });
                  }
                }}
                aria-label={t("apiKeys.shares.revokeGameShare")}
              />
            )}
          </Group>
        ))}
      </Stack>
    </Stack>
  );
}
