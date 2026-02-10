import { useState } from "react";
import {
  Stack,
  Group,
  Text,
  Collapse,
  UnstyledButton,
  Badge,
} from "@mantine/core";
import {
  IconChevronDown,
  IconChevronRight,
  IconBuilding,
  IconHeartFilled,
  IconExternalLink,
  IconLink,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import { DeleteIconButton } from "@components/buttons";
import { useRemoveGameSponsor, useRevokePrivateShare } from "@/api/hooks";
import type { ObjApiKeyShare } from "@/api/generated";

interface ApiKeySharesProps {
  shares: ObjApiKeyShare[];
}

export function ApiKeyShares({ shares }: ApiKeySharesProps) {
  const { t } = useTranslation("common");
  const [opened, setOpened] = useState(false);

  // Show institution/workshop shares (self-shares with only user set are excluded)
  const orgShares = shares.filter((s) => s.institution || s.workshop);
  const publicSponsorships = shares.filter((s) => s.game && !s.isPrivateShare);
  const privateShares = shares.filter((s) => s.game && s.isPrivateShare);

  const totalShares =
    orgShares.length + publicSponsorships.length + privateShares.length;

  if (totalShares === 0) return null;

  return (
    <Stack gap={0}>
      <UnstyledButton onClick={() => setOpened((o) => !o)} px="sm" py={6}>
        <Group gap={6} align="center">
          {opened ? (
            <IconChevronDown size={18} color="var(--mantine-color-dimmed)" />
          ) : (
            <IconChevronRight size={18} color="var(--mantine-color-dimmed)" />
          )}
          <Text size="sm" c="dimmed" fw={500}>
            {t("apiKeys.shares.toggle")}
          </Text>
          {!opened && totalShares > 0 && (
            <Badge size="sm" variant="light" color="gray">
              {totalShares}
            </Badge>
          )}
        </Group>
      </UnstyledButton>

      <Collapse in={opened}>
        <Stack gap="sm" px="sm" pb="sm">
          {orgShares.length > 0 && <OrgSharesSection shares={orgShares} />}
          {publicSponsorships.length > 0 && (
            <SponsorshipsSection shares={publicSponsorships} />
          )}
          {privateShares.length > 0 && (
            <PrivateSharesSection shares={privateShares} />
          )}
        </Stack>
      </Collapse>
    </Stack>
  );
}

function OrgSharesSection({ shares }: { shares: ObjApiKeyShare[] }) {
  const { t } = useTranslation("common");
  const navigate = useNavigate();

  return (
    <Stack gap={4}>
      <Group gap={8} align="center">
        <IconBuilding size={18} />
        <Text size="sm" fw={600} c="dimmed" tt="uppercase">
          {t("apiKeys.shares.orgShares")} ({shares.length})
        </Text>
      </Group>
      <Stack gap={6} pl="md">
        {shares.map((share) => {
          const name =
            share.institution?.name ||
            share.workshop?.name ||
            share.user?.name ||
            "-";
          return (
            <Group key={share.id} gap="sm" align="center">
              <Text size="sm" fw={500}>
                {name}
              </Text>
              {share.institution?.id && (
                <UnstyledButton
                  onClick={() => navigate({ to: "/my-organization" })}
                >
                  <Group gap={4} align="center">
                    <IconExternalLink
                      size={16}
                      color="var(--mantine-color-accent-5)"
                    />
                    <Text size="sm" c="accent.5" fw={500}>
                      {t("apiKeys.shares.goToOrg")}
                    </Text>
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

function SponsorshipsSection({ shares }: { shares: ObjApiKeyShare[] }) {
  const { t } = useTranslation("common");
  const removeGameSponsor = useRemoveGameSponsor();

  return (
    <Stack gap={4}>
      <Group gap={8} align="center">
        <IconHeartFilled size={18} color="var(--mantine-color-pink-5)" />
        <Text size="sm" fw={600} c="dimmed" tt="uppercase">
          {t("apiKeys.shares.sponsorships")} ({shares.length})
        </Text>
      </Group>
      <Stack gap={6} pl="md">
        {shares.map((share) => (
          <Group key={share.id} gap="sm" align="center" justify="space-between">
            <Text size="sm">
              {share.game?.name || t("apiKeys.shares.unknownGame")}
            </Text>
            <DeleteIconButton
              size="sm"
              onClick={() => {
                if (share.game?.id) {
                  removeGameSponsor.mutate(share.game.id);
                }
              }}
              aria-label={t("apiKeys.shares.removeSponsor")}
            />
          </Group>
        ))}
      </Stack>
    </Stack>
  );
}

function PrivateSharesSection({ shares }: { shares: ObjApiKeyShare[] }) {
  const { t } = useTranslation("common");
  const revokePrivateShare = useRevokePrivateShare();

  return (
    <Stack gap={4}>
      <Group gap={8} align="center">
        <IconLink size={18} color="var(--mantine-color-accent-5)" />
        <Text size="sm" fw={600} c="dimmed" tt="uppercase">
          {t("apiKeys.shares.privateShares")} ({shares.length})
        </Text>
      </Group>
      <Stack gap={6} pl="md">
        {shares.map((share) => (
          <Group key={share.id} gap="sm" align="center" justify="space-between">
            <Text size="sm">
              {share.game?.name || t("apiKeys.shares.unknownGame")}
            </Text>
            <DeleteIconButton
              size="sm"
              onClick={() => {
                if (share.game?.id) {
                  revokePrivateShare.mutate(share.game.id);
                }
              }}
              aria-label={t("apiKeys.shares.revokePrivateShare")}
            />
          </Group>
        ))}
      </Stack>
    </Stack>
  );
}
