/**
 * ThemePickerModal - Fullscreen modal for selecting and previewing game themes.
 *
 * Layout (all screen sizes):
 * - Preview fills the entire screen
 * - Settings button in the modal title bar opens a right-side drawer
 *   with preset, animation, and thinking text controls
 * - Action buttons (Apply / Remove / Cancel) live in the drawer footer
 */

import {
  Modal,
  Stack,
  Select,
  TextInput,
  Group,
  Text,
  Box,
  ScrollArea,
  Drawer,
} from "@mantine/core";
import { IconSettings } from "@tabler/icons-react";
import { useDisclosure, useMediaQuery } from "@mantine/hooks";
import { useTranslation } from "react-i18next";
import { useState, useEffect, useMemo } from "react";
import { ActionButton, CancelButton, DangerButton } from "@components/buttons";
import { PRESETS } from "@/features/game-player-v2/theme";
import type { BackgroundAnimation } from "@/features/game-player-v2/theme";
import type { ObjGameTheme } from "@/api/generated";
import { ThemePreview } from "./ThemePreview";

/** All available background animations */
const ANIMATIONS: BackgroundAnimation[] = [
  "none",
  "stars",
  "bubbles",
  "fireflies",
  "snow",
  "bits",
  "matrixRain",
  "embers",
  "hyperspace",
  "sparkles",
  "hearts",
  "glitch",
  "circuits",
  "leaves",
  "geometric",
  "confetti",
  "confettiExplosion",
  "waves",
  "glowworm",
  "sun",
  "tumbleweed",
];

interface ThemePickerModalProps {
  opened: boolean;
  onClose: () => void;
  /** Current theme value (null = no theme / AI-generated) */
  value: ObjGameTheme | null | undefined;
  /** Called when user applies a theme selection */
  onChange: (theme: ObjGameTheme | null) => void;
  /** If true, the user can only view, not edit */
  readOnly?: boolean;
}

export function ThemePickerModal({
  opened,
  onClose,
  value,
  onChange,
  readOnly = false,
}: ThemePickerModalProps) {
  const { t } = useTranslation("common");
  const isMobile = useMediaQuery("(max-width: 48em)");
  const [drawerOpened, { open: openDrawer, close: closeDrawer }] =
    useDisclosure(false);

  // Local draft state (not saved until Apply)
  const [preset, setPreset] = useState<string>("default");
  const [animation, setAnimation] = useState<string>("");
  const [thinkingText, setThinkingText] = useState<string>("");

  // Sync local state from prop when modal opens
  /* eslint-disable react-hooks/set-state-in-effect -- Intentional: initialize form from prop data */
  useEffect(() => {
    if (opened) {
      setPreset(value?.preset || "default");
      setAnimation(value?.animation || "");
      setThinkingText(value?.thinkingText || "");
    }
  }, [opened, value]);

  // Build preset options — translated and sorted alphabetically
  const presetOptions = useMemo(
    () =>
      Object.keys(PRESETS)
        .map((key) => ({
          value: key,
          label: t(
            `games.theme.presets.${key}`,
            key.charAt(0).toUpperCase() + key.slice(1),
          ),
        }))
        .sort((a, b) => a.label.localeCompare(b.label)),
    [t],
  );

  // Build animation options — translated and sorted, with "Preset default" first
  const animationOptions = useMemo(
    () => [
      { value: "", label: t("games.theme.animationDefault") },
      ...ANIMATIONS.map((a) => ({
        value: a,
        label: t(
          `games.theme.animations.${a}`,
          a.charAt(0).toUpperCase() + a.slice(1),
        ),
      })).sort((a, b) => a.label.localeCompare(b.label)),
    ],
    [t],
  );

  // Build live preview theme from current draft
  const previewTheme: ObjGameTheme = useMemo(
    () => ({
      preset,
      animation: animation || undefined,
      thinkingText: thinkingText || undefined,
    }),
    [preset, animation, thinkingText],
  );

  const handleApply = () => {
    onChange({
      preset,
      animation: animation || undefined,
      thinkingText: thinkingText || undefined,
    });
    onClose();
  };

  const handleRemove = () => {
    onChange(null);
    onClose();
  };

  // Controls (reused in sidebar and drawer)
  const controls = (
    <Stack gap="md">
      <Select
        label={t("games.theme.preset")}
        description={t("games.theme.presetDescription")}
        data={presetOptions}
        value={preset}
        onChange={(v) => v && setPreset(v)}
        searchable
        readOnly={readOnly}
        maxDropdownHeight={400}
      />
      <Select
        label={t("games.theme.animation")}
        description={t("games.theme.animationDescription")}
        data={animationOptions}
        value={animation}
        onChange={(v) => setAnimation(v ?? "")}
        searchable
        clearable
        readOnly={readOnly}
      />
      <TextInput
        label={t("games.theme.thinkingText")}
        description={t("games.theme.thinkingTextDescription")}
        placeholder={t("games.theme.thinkingTextPlaceholder")}
        value={thinkingText}
        onChange={(e) => setThinkingText(e.target.value)}
        readOnly={readOnly}
      />
    </Stack>
  );

  // Action buttons (reused in sidebar and drawer)
  const actions = (
    <>
      {!readOnly ? (
        <Stack gap="sm">
          <Group justify="flex-end" gap="sm">
            <CancelButton onClick={onClose}>{t("cancel")}</CancelButton>
            <ActionButton onClick={handleApply}>
              {t("games.theme.apply")}
            </ActionButton>
          </Group>
          {value && (
            <DangerButton onClick={handleRemove} fullWidth>
              {t("games.theme.remove")}
            </DangerButton>
          )}
        </Stack>
      ) : (
        <Group justify="flex-end">
          <CancelButton onClick={onClose}>{t("close")}</CancelButton>
        </Group>
      )}
    </>
  );

  return (
    <>
      <Modal
        opened={opened}
        onClose={onClose}
        title={
          <Group gap="sm">
            <Text fw={600}>{t("games.theme.modalTitle")}</Text>
            {/* Settings button only on mobile — desktop has inline sidebar */}
            {isMobile && (
              <ActionButton
                leftSection={<IconSettings size={16} />}
                onClick={openDrawer}
                size="compact-sm"
              >
                {t("games.theme.changeDesign")}
              </ActionButton>
            )}
          </Group>
        }
        fullScreen
        styles={{
          body: {
            height: "calc(100vh - 60px)",
            padding: 0,
            overflow: "hidden",
          },
        }}
      >
        {isMobile ? (
          /* Mobile: preview fills the entire modal */
          <ThemePreview apiTheme={previewTheme} />
        ) : (
          /* Desktop: preview left, controls sidebar right */
          <Box style={{ display: "flex", height: "100%" }}>
            <Box style={{ flex: 1, minWidth: 0 }}>
              <ThemePreview apiTheme={previewTheme} />
            </Box>
            <Box
              style={{
                width: 340,
                flexShrink: 0,
                borderLeft: "1px solid var(--mantine-color-gray-3)",
                display: "flex",
                flexDirection: "column",
              }}
            >
              <ScrollArea style={{ flex: 1 }} p="md">
                {controls}
              </ScrollArea>
              <Box
                p="md"
                style={{ borderTop: "1px solid var(--mantine-color-gray-3)" }}
              >
                {actions}
              </Box>
            </Box>
          </Box>
        )}
      </Modal>

      {/* Mobile-only: settings drawer overlays on top */}
      <Drawer
        opened={drawerOpened}
        onClose={closeDrawer}
        position="right"
        size="sm"
        title={<Text fw={600}>{t("games.theme.modalTitle")}</Text>}
        styles={{
          body: {
            display: "flex",
            flexDirection: "column",
            height: "calc(100% - 60px)",
          },
        }}
      >
        {controls}
        <Box mt="auto" pt="md">
          {actions}
        </Box>
      </Drawer>
    </>
  );
}
