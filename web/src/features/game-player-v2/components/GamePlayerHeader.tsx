import {
  Group,
  Text,
  ActionIcon,
  Tooltip,
  Box,
  Menu,
  Checkbox,
} from "@mantine/core";
import { useTranslation } from "react-i18next";
import {
  IconArrowLeft,
  IconTextIncrease,
  IconTextDecrease,
  IconTypography,
  IconSettings,
  IconRestore,
} from "@tabler/icons-react";
import env from "@/config/env";
import type { FontSize } from "../context";
import type { PartialGameTheme } from "../theme/types";
import { useGameTheme } from "../theme";
import { ThemeTestPanel } from "./ThemeTestPanel";
import classes from "./GamePlayer.module.css";

interface GamePlayerHeaderProps {
  gameName?: string;
  gameDescription?: string;

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
  debugMode: boolean;
  onToggleDebugMode: () => void;

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
  debugMode,
  onToggleDebugMode,
  onBack,
  currentTheme,
  onThemeChange,
}: GamePlayerHeaderProps) {
  const { t } = useTranslation("common");

  return (
    <HeaderWithTheme>
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
              <Menu.Divider />
              <Menu.Item closeMenuOnClick={false} onClick={onToggleDebugMode}>
                <Checkbox
                  label={t("gamePlayer.header.debug")}
                  checked={debugMode}
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
