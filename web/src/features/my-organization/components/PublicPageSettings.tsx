import { useState } from "react";
import {
  ActionIcon,
  Alert,
  Anchor,
  Button,
  Card,
  Group,
  Stack,
  Switch,
  Text,
  TextInput,
  Textarea,
  Tooltip,
} from "@mantine/core";
import {
  IconAlertTriangle,
  IconDeviceFloppy,
  IconExternalLink,
  IconQrcode,
  IconWorld,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useUpdateWorkshop } from "@/api/hooks";
import type { ObjWorkshop } from "@/api/generated";
import { ShareLinkModal } from "@components/share";
import { buildShareUrl } from "@/common/lib/url";
import { ErrorCodes, extractRawErrorCode } from "@/common/types/errorCodes";
import {
  PUBLIC_DESCRIPTION_MAX_LENGTH,
  PUBLIC_SLUG_MAX_LENGTH,
  PUBLIC_WORKSHOP_PREFIX,
  isValidPublicSlug,
  normalizePublicSlug,
  publicWorkshopPath,
} from "@/common/lib/publicWorkshop";
import { ConfirmationModal } from "./ConfirmationModal";

interface PublicPageSettingsProps {
  workshop: ObjWorkshop;
  /** Called after each save, e.g. to refresh settings embedded elsewhere. */
  onSaved?: () => void;
}

/** The "Öffentliche Seite" section of the workshop settings. */
export function PublicPageSettings({ workshop, onSaved }: PublicPageSettingsProps) {
  const { t } = useTranslation("common");
  const updateWorkshop = useUpdateWorkshop();
  const savedSlug = workshop.publicSlug ?? "";
  const [slugInput, setSlugInput] = useState(savedSlug);
  const [slugError, setSlugError] = useState<string | null>(null);
  const [descriptionError, setDescriptionError] = useState<string | null>(null);
  const [confirmSlugOpen, setConfirmSlugOpen] = useState(false);
  const [shareOpen, setShareOpen] = useState(false);

  const slug = normalizePublicSlug(slugInput);
  const slugChanged = slug !== savedSlug;
  const url = buildShareUrl(publicWorkshopPath(savedSlug));

  const save = async (
    changes: Partial<{
      public: boolean;
      publicSlug: string;
      publicDescription: string;
    }>,
  ) => {
    if (!workshop.id) return;
    await updateWorkshop.mutateAsync({
      id: workshop.id,
      name: workshop.name || "",
      active: workshop.active || false,
      public: changes.public ?? workshop.public ?? false,
      showPublicGames: workshop.showPublicGames ?? false,
      showOtherParticipantsGames: workshop.showOtherParticipantsGames ?? true,
      designEditingEnabled: workshop.designEditingEnabled ?? false,
      aiQualityTier: workshop.aiQualityTier ?? undefined,
      promptConstraints: workshop.promptConstraints ?? undefined,
      isPaused: workshop.isPaused ?? false,
      allowGameSharing: workshop.allowGameSharing ?? false,
      publicSlug: changes.publicSlug,
      publicDescription: changes.publicDescription,
    });
    onSaved?.();
  };

  const saveSlug = async () => {
    setConfirmSlugOpen(false);
    try {
      await save({ publicSlug: slug });
      setSlugInput(slug);
      setSlugError(null);
    } catch (error) {
      setSlugError(
        extractRawErrorCode(error) === ErrorCodes.PUBLIC_SLUG_TAKEN
          ? t("myOrganization.publicPage.slugTaken")
          : t("myOrganization.publicPage.slugInvalid"),
      );
    }
  };

  const requestSlugSave = () => {
    if (!isValidPublicSlug(slug)) {
      setSlugError(t("myOrganization.publicPage.slugInvalid"));
      return;
    }
    setSlugError(null);
    setConfirmSlugOpen(true);
  };

  const saveDescription = async (value: string) => {
    if (value.trim() === (workshop.publicDescription ?? "")) return;
    try {
      await save({ publicDescription: value });
      setDescriptionError(null);
    } catch {
      setDescriptionError(t("myOrganization.publicPage.descriptionError"));
    }
  };

  return (
    <Card
      padding="md"
      radius="md"
      withBorder
      bg="var(--mantine-color-blue-0)"
      style={{ borderColor: "var(--mantine-color-blue-3)" }}
    >
      <Stack gap="sm">
        <Group gap="xs">
          <IconWorld size={16} color="var(--mantine-color-blue-7)" />
          <Text size="sm" fw={600} c="blue.8">
            {t("myOrganization.publicPage.title")}
          </Text>
        </Group>
        <Text size="xs" c="dimmed">
          {t("myOrganization.publicPage.description")}
        </Text>
        <Alert
          color="yellow"
          variant="light"
          icon={<IconAlertTriangle size={16} />}
          p="xs"
        >
          <Text size="xs">{t("games.privacyHint")}</Text>
        </Alert>

        <Switch
          size="sm"
          label={t("myOrganization.publicPage.enabled")}
          description={t("myOrganization.publicPage.enabledHint")}
          checked={workshop.public || false}
          disabled={updateWorkshop.isPending}
          onChange={(e) => save({ public: e.currentTarget.checked })}
        />

        <Stack gap={4}>
          <Text size="sm" fw={500}>
            {t("myOrganization.publicPage.link")}
          </Text>
          <Group gap="xs" wrap="nowrap" align="flex-start">
            <TextInput
              size="sm"
              style={{ flex: 1, minWidth: 0 }}
              leftSection={
                <Text size="xs" c="dimmed" pl={8} style={{ whiteSpace: "nowrap" }}>
                  {PUBLIC_WORKSHOP_PREFIX}
                </Text>
              }
              leftSectionWidth={32}
              value={slugInput}
              maxLength={PUBLIC_SLUG_MAX_LENGTH}
              onChange={(e) => {
                setSlugInput(e.currentTarget.value);
                setSlugError(null);
              }}
              error={slugError}
              autoCapitalize="none"
              autoCorrect="off"
              spellCheck={false}
            />
            {slugChanged && (
              <Tooltip label={t("save")}>
                <ActionIcon
                  size="lg"
                  variant="filled"
                  onClick={requestSlugSave}
                  loading={updateWorkshop.isPending}
                  aria-label={t("save")}
                >
                  <IconDeviceFloppy size={18} />
                </ActionIcon>
              </Tooltip>
            )}
          </Group>
          <Text size="xs" c="dimmed">
            {t("myOrganization.publicPage.linkHint")}
          </Text>
        </Stack>

        <Textarea
          size="sm"
          label={t("myOrganization.publicPage.text")}
          description={t("myOrganization.publicPage.textHint")}
          minRows={3}
          maxRows={10}
          autosize
          maxLength={PUBLIC_DESCRIPTION_MAX_LENGTH}
          defaultValue={workshop.publicDescription || ""}
          key={`public-description-${workshop.id}-${workshop.publicDescription || ""}`}
          disabled={updateWorkshop.isPending}
          error={descriptionError}
          onBlur={(e) => saveDescription(e.currentTarget.value)}
        />

        <Group gap="sm">
          <Button
            size="xs"
            variant="light"
            leftSection={<IconQrcode size={16} />}
            onClick={() => setShareOpen(true)}
            disabled={!savedSlug}
          >
            {t("myOrganization.publicPage.showLink")}
          </Button>
          {workshop.public && savedSlug && (
            <Anchor href={url} target="_blank" rel="noopener" size="xs">
              <Group gap={4}>
                <IconExternalLink size={14} />
                {t("myOrganization.publicPage.open")}
              </Group>
            </Anchor>
          )}
        </Group>
        {!workshop.public && (
          <Text size="xs" c="dimmed">
            {t("myOrganization.publicPage.offHint")}
          </Text>
        )}
      </Stack>

      <ConfirmationModal
        opened={confirmSlugOpen}
        onClose={() => setConfirmSlugOpen(false)}
        onConfirm={saveSlug}
        title={t("myOrganization.publicPage.changeLinkTitle")}
        message={t("myOrganization.publicPage.changeLinkMessage", {
          url: buildShareUrl(publicWorkshopPath(slug)),
        })}
        warning={t("myOrganization.publicPage.changeLinkWarning")}
        confirmIcon={<IconDeviceFloppy size={16} />}
        confirmColor="blue"
        isLoading={updateWorkshop.isPending}
      />

      <ShareLinkModal
        opened={shareOpen}
        onClose={() => setShareOpen(false)}
        title={t("myOrganization.publicPage.shareTitle")}
        description={t("myOrganization.publicPage.shareDescription")}
        url={url}
      />
    </Card>
  );
}
