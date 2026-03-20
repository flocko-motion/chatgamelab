import { useState } from "react";
import {
  Group,
  Text,
  ActionIcon,
  Tooltip,
  Box,
  Menu,
  Checkbox,
  Badge,
  Modal,
  Stack,
  Code,
  CopyButton,
  Button,
} from "@mantine/core";
import { useTranslation } from "react-i18next";
import {
  IconArrowLeft,
  IconVolume,
  IconVolumeOff,
  IconTextIncrease,
  IconTextDecrease,
  IconTypography,
  IconSettings,
  IconRestore,
  IconCopy,
  IconCheck,
} from "@tabler/icons-react";
import env, { config } from "@/config/env";
import { getNativeLanguageName } from "@/i18n/languages";
import { useAuth } from "@/providers/AuthProvider";
import type { FontSize } from "../context";
import type { PartialGameTheme } from "../theme/types";
import { useGameTheme } from "../theme";
import { ThemeTestPanel } from "./ThemeTestPanel";
import classes from "./GamePlayer.module.css";

interface GamePlayerHeaderProps {
  gameName?: string;
  gameDescription?: string;
  sessionLanguage?: string | null;

  // IDs for debug modal
  sessionId?: string | null;
  gameId?: string | null;
  messageCount?: number;

  // AI info
  aiModel?: string | null;
  aiPlatform?: string | null;
  hasAudioOut?: boolean;
  isAudioMuted: boolean;
  onToggleAudioMuted: () => void;

  // Font size
  fontSize: FontSize;
  increaseFontSize: () => void;
  decreaseFontSize: () => void;
  resetFontSize: () => void;

  // Settings toggles
  animationEnabled: boolean;
  onToggleAnimation: () => void;
  textEffectsEnabled: boolean;
  onToggleTextEffects: () => void;
  useNeutralTheme: boolean;
  onToggleNeutralTheme: () => void;

  // Navigation
  onBack: () => void;

  // Dev theme panel
  currentTheme?: PartialGameTheme;
  onThemeChange: (theme: PartialGameTheme) => void;
}

/** Header with theme-aware styling */
function HeaderWithTheme({ children }: { children: React.ReactNode }) {
  const { cssVars } = useGameTheme();

  return (
    <Box className={classes.header} px="md" py="sm" style={cssVars}>
      {children}
    </Box>
  );
}

export function GamePlayerHeader({
  gameName,
  gameDescription,
  sessionLanguage,
  sessionId,
  gameId,
  messageCount,
  aiModel,
  aiPlatform,
  hasAudioOut,
  isAudioMuted,
  onToggleAudioMuted,
  fontSize,
  increaseFontSize,
  decreaseFontSize,
  resetFontSize,
  animationEnabled,
  onToggleAnimation,
  textEffectsEnabled,
  onToggleTextEffects,
  useNeutralTheme,
  onToggleNeutralTheme,
  onBack,
  currentTheme,
  onThemeChange,
}: GamePlayerHeaderProps) {
  const { t } = useTranslation("common");
  const { backendUser } = useAuth();
  const sessionLanguageLabel = getNativeLanguageName(sessionLanguage);
  const [debugModalOpen, setDebugModalOpen] = useState(false);

  const debugInfo = buildDebugInfo({
    userId: backendUser?.id,
    userEmail: backendUser?.email,
    gameId: gameId ?? undefined,
    sessionId: sessionId ?? undefined,
    messageCount,
    aiModel: aiModel ?? undefined,
    aiPlatform: aiPlatform ?? undefined,
    sessionLanguage: sessionLanguage ?? undefined,
  });

  return (
    <HeaderWithTheme>
      <SessionDebugModal
        opened={debugModalOpen}
        onClose={() => setDebugModalOpen(false)}
        info={debugInfo}
      />
      <Group justify="space-between" wrap="nowrap">
        <Group gap="sm" wrap="nowrap" style={{ minWidth: 0, flex: 1 }}>
          <Tooltip label={t("gamePlayer.header.back")} position="bottom">
            <ActionIcon
              variant="subtle"
              color="gray"
              onClick={onBack}
              aria-label={t("gamePlayer.header.back")}
              size="lg"
            >
              <IconArrowLeft size={20} />
            </ActionIcon>
          </Tooltip>
          <Box style={{ minWidth: 0, flex: 1 }}>
            <Text fw={600} truncate size="sm">
              {gameName || t("gamePlayer.unnamed")}
            </Text>
            {gameDescription && (
              <Text size="xs" truncate className={classes.headerDescription}>
                {gameDescription}
              </Text>
            )}
          </Box>
        </Group>
        <Group gap="xs" wrap="nowrap">
          {sessionLanguageLabel && (
            <Badge
              variant="light"
              color="gray"
              size="sm"
              style={{ flexShrink: 0 }}
            >
              {sessionLanguageLabel}
            </Badge>
          )}
          {(aiPlatform || aiModel) && (
            <Badge
              variant="light"
              color="gray"
              size="sm"
              style={{ flexShrink: 0, cursor: "pointer" }}
              onClick={() => setDebugModalOpen(true)}
            >
              {[aiPlatform, aiModel].filter(Boolean).join(" / ")}
            </Badge>
          )}
          {hasAudioOut && (
            <Tooltip
              label={
                isAudioMuted
                  ? t("gamePlayer.header.unmuteAudio")
                  : t("gamePlayer.header.muteAudio")
              }
              position="bottom"
            >
              <ActionIcon
                variant="subtle"
                color={isAudioMuted ? "violet" : "gray"}
                aria-label={
                  isAudioMuted
                    ? t("gamePlayer.header.unmuteAudio")
                    : t("gamePlayer.header.muteAudio")
                }
                size="lg"
                onClick={onToggleAudioMuted}
              >
                {isAudioMuted ? (
                  <IconVolumeOff size={18} />
                ) : (
                  <IconVolume size={18} />
                )}
              </ActionIcon>
            </Tooltip>
          )}
          <Menu shadow="md" width={200} position="bottom-end">
            <Menu.Target>
              <Tooltip
                label={t("gamePlayer.header.fontSize")}
                position="bottom"
              >
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  aria-label={t("gamePlayer.header.fontSize")}
                  size="lg"
                >
                  <IconTypography size={18} />
                </ActionIcon>
              </Tooltip>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Label>{t("gamePlayer.header.fontSize")}</Menu.Label>
              <Menu.Item
                leftSection={<IconTextIncrease size={16} />}
                onClick={increaseFontSize}
                disabled={fontSize === "3xl"}
              >
                {t("gamePlayer.header.increaseFont")}
              </Menu.Item>
              <Menu.Item
                leftSection={<IconTextDecrease size={16} />}
                onClick={decreaseFontSize}
                disabled={fontSize === "xs"}
              >
                {t("gamePlayer.header.decreaseFont")}
              </Menu.Item>
              <Menu.Item
                leftSection={<IconRestore size={16} />}
                onClick={resetFontSize}
                disabled={fontSize === "md"}
              >
                {t("gamePlayer.header.resetFont")}
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>
          <Menu shadow="md" width={200} position="bottom-end">
            <Menu.Target>
              <Tooltip
                label={t("gamePlayer.header.settings")}
                position="bottom"
              >
                <ActionIcon
                  variant="subtle"
                  color="gray"
                  aria-label={t("gamePlayer.header.settings")}
                  size="lg"
                >
                  <IconSettings size={18} />
                </ActionIcon>
              </Tooltip>
            </Menu.Target>
            <Menu.Dropdown>
              <Menu.Label>{t("gamePlayer.header.settings")}</Menu.Label>
              <Menu.Item closeMenuOnClick={false} onClick={onToggleAnimation}>
                <Checkbox
                  label={t("gamePlayer.header.disableAnimations")}
                  checked={!animationEnabled}
                  readOnly
                  tabIndex={-1}
                  size="sm"
                  styles={{
                    root: { pointerEvents: "none" },
                    input: { cursor: "pointer" },
                    label: { cursor: "pointer" },
                  }}
                />
              </Menu.Item>
              <Menu.Item closeMenuOnClick={false} onClick={onToggleTextEffects}>
                <Checkbox
                  label={t("gamePlayer.header.disableTextEffects")}
                  checked={!textEffectsEnabled}
                  readOnly
                  tabIndex={-1}
                  size="sm"
                  styles={{
                    root: { pointerEvents: "none" },
                    input: { cursor: "pointer" },
                    label: { cursor: "pointer" },
                  }}
                />
              </Menu.Item>
              <Menu.Item
                closeMenuOnClick={false}
                onClick={onToggleNeutralTheme}
              >
                <Checkbox
                  label={t("gamePlayer.header.useNeutralTheme")}
                  checked={useNeutralTheme}
                  readOnly
                  tabIndex={-1}
                  size="sm"
                  styles={{
                    root: { pointerEvents: "none" },
                    input: { cursor: "pointer" },
                    label: { cursor: "pointer" },
                  }}
                />
              </Menu.Item>
            </Menu.Dropdown>
          </Menu>
          {env.DEV && currentTheme && (
            <ThemeTestPanel
              currentTheme={currentTheme}
              onThemeChange={onThemeChange}
            />
          )}
        </Group>
      </Group>
    </HeaderWithTheme>
  );
}

// ── Session Debug Modal ─────────────────────────────────────────────────

interface DebugInfo {
  lines: { label: string; value: string }[];
  clipboardText: string;
}

function buildDebugInfo(data: {
  userId?: string;
  userEmail?: string;
  gameId?: string;
  sessionId?: string;
  messageCount?: number;
  aiModel?: string;
  aiPlatform?: string;
  sessionLanguage?: string;
}): DebugInfo {
  const now = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  const datetime = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`;

  // Extract hostname from API base URL (e.g. "localhost:7001" or "prod.example.com")
  const server = (() => {
    try { return new URL(config.API_BASE_URL).host; } catch { return config.API_BASE_URL; }
  })();

  const lines: { label: string; value: string }[] = [
    { label: "Server", value: server },
    { label: "Date/Time", value: datetime },
    { label: "Session ID", value: data.sessionId || "—" },
    { label: "Game ID", value: data.gameId || "—" },
    { label: "User ID", value: data.userId || "—" },
    { label: "User Email", value: data.userEmail || "—" },
    { label: "AI Platform", value: data.aiPlatform || "—" },
    { label: "AI Model", value: data.aiModel || "—" },
    { label: "Language", value: data.sessionLanguage || "—" },
    { label: "Messages", value: data.messageCount != null ? String(data.messageCount) : "—" },
  ];

  const clipboardText = lines
    .filter(l => l.value !== "—")
    .map(l => `${l.label}: ${l.value}`)
    .join('\n');

  return { lines, clipboardText };
}

function SessionDebugModal({
  opened,
  onClose,
  info,
}: {
  opened: boolean;
  onClose: () => void;
  info: DebugInfo;
}) {
  return (
    <Modal
      opened={opened}
      onClose={onClose}
      title="Session Details"
      size="md"
      centered
    >
      <Stack gap="xs">
        {info.lines.map((line) => (
          <Group key={line.label} justify="space-between" wrap="nowrap">
            <Text size="sm" c="dimmed" style={{ flexShrink: 0 }}>
              {line.label}
            </Text>
            <Code style={{ fontSize: 'var(--mantine-font-size-xs)' }}>
              {line.value}
            </Code>
          </Group>
        ))}
        <CopyButton value={info.clipboardText}>
          {({ copied, copy }) => (
            <Button
              variant="light"
              color={copied ? "teal" : "gray"}
              leftSection={copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
              onClick={copy}
              fullWidth
              mt="sm"
            >
              {copied ? "Copied!" : "Copy for bug report"}
            </Button>
          )}
        </CopyButton>
      </Stack>
    </Modal>
  );
}
