import { Badge, Center, Group, Loader, Modal, NumberInput, Stack } from "@mantine/core";
import { IconLink } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton, DeleteIconButton, EditIconButton, TextButton } from "@components/buttons";
import { BodyText, HelperText, Label } from "@components/typography";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import { useOrgKeyGameShares } from "../hooks/useOrgKeyGameShares";

/** The game share links that play on one of the organisation's keys. */
export function OrgKeyDetails({ shareId }: { shareId: string }) {
  const { t } = useTranslation("common");
  const { isMobile } = useResponsiveDesign();
  const state = useOrgKeyGameShares(shareId);
  const { editModal } = state;

  if (state.isLoading) {
    return (
      <Center py="sm">
        <Loader size="sm" />
      </Center>
    );
  }

  if (state.gameShares.length === 0) {
    return <HelperText>{t("apiKeys.shares.noSponsorships")}</HelperText>;
  }

  return (
    <>
      <Stack gap={4}>
        <Group gap={8} align="center">
          <IconLink size={18} color="var(--mantine-color-accent-5)" />
          <Label uppercase>
            {t("apiKeys.shares.gameShares")} ({state.gameShares.length})
          </Label>
        </Group>
        <Stack gap={6} pl="md">
          {state.gameShares.map((gameShare) => (
            <Group key={gameShare.id} gap="sm" align="center" justify="space-between" wrap="nowrap">
              <Stack gap={2} style={{ minWidth: 0, flex: 1 }}>
                <Group gap="xs" wrap="wrap">
                  <BodyText size="sm" fw={500}>
                    {gameShare.gameName || t("apiKeys.shares.unknownGame")}
                  </BodyText>
                  <Badge size="xs" variant="light" color="gray">
                    {state.contextLabel(gameShare)}
                  </Badge>
                </Group>
                <HelperText>{state.remainingLabel(gameShare)}</HelperText>
              </Stack>
              <Group gap={4} wrap="nowrap" style={{ flexShrink: 0 }}>
                <EditIconButton
                  size="sm"
                  onClick={() => state.startEdit(gameShare)}
                  aria-label={t("apiKeys.shares.editRemaining")}
                />
                <DeleteIconButton
                  size="sm"
                  onClick={() => state.revoke(gameShare)}
                  aria-label={t("apiKeys.shares.revokeGameShare")}
                />
              </Group>
            </Group>
          ))}
        </Stack>
      </Stack>

      <Modal
        opened={editModal.opened}
        onClose={editModal.close}
        title={t("apiKeys.shares.editRemaining")}
        size="sm"
        fullScreen={isMobile}
        centered={!isMobile}
      >
        <Stack gap="md">
          <NumberInput
            label={t("games.privateShare.maxSessions")}
            description={t("games.privateShare.maxSessionsDescription")}
            placeholder={t("apiKeys.shares.unlimited")}
            value={editModal.maxSessions}
            onChange={(v) => editModal.setMaxSessions(v === "" ? "" : v)}
            min={1}
            allowNegative={false}
            allowDecimal={false}
          />
          <Group justify="flex-end" gap="sm">
            <TextButton onClick={editModal.close}>{t("cancel")}</TextButton>
            <ActionButton onClick={editModal.submit} loading={editModal.loading} size="sm">
              {t("save")}
            </ActionButton>
          </Group>
        </Stack>
      </Modal>
    </>
  );
}
