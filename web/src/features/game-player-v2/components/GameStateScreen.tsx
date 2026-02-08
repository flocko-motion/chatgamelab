import { Group, Text, Loader, Box, Stack } from "@mantine/core";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { IconAlertCircle, IconArrowLeft, IconKey } from "@tabler/icons-react";
import { ActionButton, TextButton } from "@components/buttons";
import { ErrorModal } from "@/common/components/ErrorModal";
import type { GamePhase } from "../types";
import classes from "./GamePlayer.module.css";

interface GameStateScreenProps {
  phase: GamePhase;
  isContinuation: boolean;
  isInWorkshopContext: boolean;
  gameLoading: boolean;
  gameError: unknown;
  gameExists: boolean;
  missingFields: string[];
  isNoApiKeyError: boolean;
  error: string | null;
  errorObject: unknown;
  onBack: () => void;
}

export function GameStateScreen({
  phase,
  isContinuation,
  isInWorkshopContext,
  gameLoading,
  gameError,
  gameExists,
  missingFields,
  isNoApiKeyError,
  error,
  errorObject,
  onBack,
}: GameStateScreenProps) {
  const { t } = useTranslation("common");
  const navigate = useNavigate();

  // Loading game data or waiting for continuation session to load
  if (gameLoading || (isContinuation && phase === "idle")) {
    return (
      <Box className={classes.container}>
        <Stack
          className={classes.stateContainer}
          align="center"
          justify="center"
          gap="md"
        >
          <Loader size="lg" color="accent" />
          <Text c="dimmed">{t("gamePlayer.loading.game")}</Text>
        </Stack>
      </Box>
    );
  }

  // Game not found
  if (!isContinuation && (gameError || !gameExists)) {
    return (
      <Box className={classes.container}>
        <Stack
          className={classes.stateContainer}
          align="center"
          justify="center"
          gap="md"
        >
          <IconAlertCircle size={48} color="var(--mantine-color-red-5)" />
          <Text size="lg" fw={600}>
            {t("gamePlayer.error.gameNotFound")}
          </Text>
          <TextButton
            onClick={onBack}
            leftSection={<IconArrowLeft size={16} />}
          >
            {t("gamePlayer.error.backToGames")}
          </TextButton>
        </Stack>
      </Box>
    );
  }

  // Missing required fields
  if (!isContinuation && missingFields.length > 0) {
    return (
      <Box className={classes.container}>
        <Stack
          className={classes.stateContainer}
          align="center"
          justify="center"
          gap="md"
        >
          <IconAlertCircle size={48} color="var(--mantine-color-red-5)" />
          <Text size="lg" fw={600}>
            {t("gamePlayer.error.missingFields")}
          </Text>
          <Text c="dimmed" ta="center">
            {missingFields.join(", ")}
          </Text>
          <TextButton
            onClick={onBack}
            leftSection={<IconArrowLeft size={16} />}
          >
            {t("gamePlayer.error.backToGames")}
          </TextButton>
        </Stack>
      </Box>
    );
  }

  // No API key error
  if (isNoApiKeyError) {
    return (
      <Box className={classes.container}>
        <Stack
          className={classes.stateContainer}
          align="center"
          justify="center"
          gap="md"
        >
          <IconAlertCircle size={48} color="var(--mantine-color-red-5)" />
          <Text size="lg" fw={600}>
            {t("gamePlayer.error.noApiKey.title", "No API Key Available")}
          </Text>
          <Text c="dimmed" ta="center" maw={400}>
            {isInWorkshopContext
              ? t(
                  "gamePlayer.error.noApiKey.workshop",
                  "No API key is configured for this workshop. Please contact your workshop administrator.",
                )
              : t(
                  "gamePlayer.error.noApiKey.personal",
                  "You need to configure an API key before you can play. Go to your API Key settings to add one.",
                )}
          </Text>
          <Group gap="sm">
            <TextButton
              onClick={onBack}
              leftSection={<IconArrowLeft size={16} />}
            >
              {t("gamePlayer.error.backToGames")}
            </TextButton>
            {!isInWorkshopContext && (
              <ActionButton
                leftSection={<IconKey size={16} />}
                onClick={() => navigate({ to: "/api-keys" })}
              >
                {t("gamePlayer.error.noApiKey.goToSettings", "API Key Settings")}
              </ActionButton>
            )}
          </Group>
        </Stack>
      </Box>
    );
  }

  // Generic error state
  if (phase === "error") {
    return (
      <>
        <Box className={classes.container}>
          <Stack
            className={classes.stateContainer}
            align="center"
            justify="center"
            gap="md"
          >
            <Loader size="lg" color="accent" />
          </Stack>
        </Box>
        <ErrorModal
          opened={true}
          onClose={onBack}
          error={errorObject}
          message={!errorObject ? error || undefined : undefined}
          title={t("gamePlayer.error.sessionFailed")}
        />
      </>
    );
  }

  // Session exists but API key was deleted — auto-resolving server-side
  if (phase === "needs-api-key") {
    return (
      <Box className={classes.container}>
        <Stack
          className={classes.stateContainer}
          align="center"
          justify="center"
          gap="md"
        >
          <Loader size="lg" color="accent" />
          <Text fw={600}>{t("gamePlayer.loading.resolvingKey", "Resolving API key...")}</Text>
        </Stack>
      </Box>
    );
  }

  // Idle or starting
  if (phase === "idle" || phase === "starting") {
    return (
      <Box className={classes.container}>
        <Stack
          className={classes.stateContainer}
          align="center"
          justify="center"
          gap="md"
        >
          <Loader size="lg" color="accent" />
          <Text fw={600}>{t("gamePlayer.loading.starting")}</Text>
          <Text c="dimmed" size="sm">
            {t("gamePlayer.loading.startingHint")}
          </Text>
        </Stack>
      </Box>
    );
  }

  // No state screen needed — return null so the caller renders the game
  return null;
}
