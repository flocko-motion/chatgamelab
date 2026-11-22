import { Alert, Badge, Card, Center, Group, Loader, Select, Stack, Switch, Textarea, Tooltip } from "@mantine/core";
import { IconAlertCircle, IconKey, IconLink, IconMail, IconUser } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { GenericIconButton } from "@components/buttons";
import { CardTitle, HelperText, Label } from "@components/typography";
import { normalizeAiQualityTier } from "@/common/lib/aiQualityTier";
import { useWorkshopSettings } from "../hooks/useWorkshopSettings";
import { AutoShareConfirmModal } from "./AutoShareConfirmModal";
import { ParticipantsList } from "./ParticipantsList";
import { PublicPageSettings } from "./PublicPageSettings";
import { WorkshopInviteModals } from "./WorkshopInviteModals";

interface SingleWorkshopSettingsProps {
  workshopId: string;
  institutionId: string;
  institutionName?: string;
  /** Shown inside another frame (a modal): no card of its own. */
  embedded?: boolean;
}

/** One workshop's settings: default key, AI tier, constraints, switches, public page, participants. */
export function SingleWorkshopSettings({
  workshopId,
  institutionId,
  institutionName,
  embedded = false,
}: SingleWorkshopSettingsProps) {
  const { t } = useTranslation("common");
  const state = useWorkshopSettings(workshopId, institutionId);
  const { workshop, invites, keyChoice } = state;

  if (state.isLoading) {
    return (
      <Center py="xl">
        <Loader />
      </Center>
    );
  }

  if (state.isError || !workshop) {
    return (
      <Alert color="red" icon={<IconAlertCircle size={16} />} title={t("error")}>
        {t("myOrganization.workshops.loadError")}
      </Alert>
    );
  }

  const hasInvite = invites.hasUsableInvite(workshop);
  const linkLabel = hasInvite
    ? t("myOrganization.workshops.viewInviteLink")
    : t("myOrganization.workshops.createInviteLink");

  return (
    <>
      <Card
        shadow={embedded ? undefined : "sm"}
        padding={embedded ? 0 : "md"}
        radius="md"
        withBorder={!embedded}
      >
        {/* Workshop header info */}
        <Group justify="space-between" wrap="nowrap" gap="sm" mb="md">
          <Stack gap={2} style={{ minWidth: 0 }}>
            <CardTitle>{workshop.name}</CardTitle>
            <Group gap="xs">
              <Badge color={workshop.active ? "green" : "gray"} variant="light" size="sm">
                {workshop.active
                  ? t("myOrganization.workshops.active")
                  : t("myOrganization.workshops.inactive")}
              </Badge>
              <Group gap={4}>
                <IconUser size={14} color="var(--mantine-color-gray-6)" />
                <HelperText>{workshop.participants?.length || 0}</HelperText>
              </Group>
            </Group>
          </Stack>
          <Group gap="xs" wrap="nowrap">
            <Tooltip label={linkLabel}>
              <GenericIconButton
                icon={<IconLink size={18} />}
                color={hasInvite ? "blue" : "gray"}
                onClick={() => invites.openLink(workshop)}
                loading={invites.creatingLink}
                aria-label={linkLabel}
              />
            </Tooltip>
            <Tooltip label={t("myOrganization.workshops.inviteByEmail")}>
              <GenericIconButton
                icon={<IconMail size={18} />}
                onClick={() => invites.openEmail(workshop)}
                aria-label={t("myOrganization.workshops.inviteByEmail")}
              />
            </Tooltip>
          </Group>
        </Group>

        <Stack gap="md" pt="md" style={{ borderTop: "1px solid var(--mantine-color-gray-3)" }}>
          {/* Default API Key Section */}
          <Stack gap="xs">
            <Group gap="xs">
              <IconKey size={14} />
              <Label>{t("myOrganization.workshops.defaultApiKey")}</Label>
            </Group>
            <Select
              size="sm"
              data={keyChoice.options}
              value={workshop.defaultApiKeyShareId || ""}
              onChange={(value) => keyChoice.choose(value || null)}
              placeholder={t("myOrganization.workshops.selectApiKey")}
              aria-label={t("myOrganization.workshops.defaultApiKey")}
              clearable
              disabled={state.settingKey}
            />
            <HelperText>{t("myOrganization.workshops.defaultApiKeyHint")}</HelperText>
          </Stack>

          {/* Workshop Settings Section */}
          <Stack gap="xs">
            <Label>{t("myOrganization.workshops.settings")}</Label>
            <HelperText>{t("myOrganization.workshops.aiQualityTierHint")}</HelperText>
            <Select
              size="sm"
              label={t("aiQualityTier.label")}
              data={state.aiQualityTierOptions}
              value={normalizeAiQualityTier(workshop.aiQualityTier)}
              onChange={(value) => state.update({ aiQualityTier: value || undefined })}
              disabled={state.updating}
            />
            <HelperText>{t("aiQualityTier.newSessionsOnly")}</HelperText>
            <Textarea
              label={t("myOrganization.workshops.promptConstraintsLabel")}
              placeholder={state.promptPlaceholder}
              description={t("myOrganization.workshops.promptConstraintsHint")}
              size="sm"
              minRows={3}
              maxRows={4}
              autosize
              maxLength={200}
              defaultValue={workshop.promptConstraints || ""}
              key={`prompt-constraints-${workshop.id}-${workshop.promptConstraints || ""}`}
              disabled={state.updating}
              onBlur={(event) => state.savePromptConstraints(event.currentTarget.value)}
            />
            <Switch
              size="sm"
              label={t("myOrganization.workshops.showPublicGames")}
              checked={workshop.showPublicGames || false}
              onChange={(e) => state.update({ showPublicGames: e.currentTarget.checked })}
            />
            <Switch
              size="sm"
              label={t("myOrganization.workshops.showOtherParticipantsGames")}
              description={t("myOrganization.workshops.showOtherParticipantsGamesHint")}
              checked={workshop.showOtherParticipantsGames !== false}
              onChange={(e) =>
                state.update({ showOtherParticipantsGames: e.currentTarget.checked })
              }
            />
            <Switch
              size="sm"
              label={t("myOrganization.workshops.isPaused")}
              description={t("myOrganization.workshops.isPausedHint")}
              checked={workshop.isPaused || false}
              onChange={(e) => state.update({ isPaused: e.currentTarget.checked })}
              color="orange"
            />
            <Switch
              size="sm"
              label={t("myOrganization.workshops.allowGameSharing")}
              description={t("myOrganization.workshops.allowGameSharingHint")}
              checked={workshop.allowGameSharing || false}
              onChange={(e) => state.update({ allowGameSharing: e.currentTarget.checked })}
            />
          </Stack>

          <PublicPageSettings
            key={workshop.publicSlug}
            workshop={workshop}
            onSaved={state.onPublicPageSaved}
          />

          <ParticipantsList workshopId={workshop.id} />
        </Stack>
      </Card>

      <WorkshopInviteModals invites={invites} />

      <AutoShareConfirmModal
        opened={keyChoice.autoShareModal.opened}
        onClose={keyChoice.autoShareModal.close}
        onConfirm={keyChoice.autoShareModal.confirm}
        keyName={keyChoice.autoShareModal.keyName}
        orgName={institutionName ?? institutionId}
        isLoading={keyChoice.autoShareModal.loading}
        error={keyChoice.autoShareModal.error}
      />
    </>
  );
}
