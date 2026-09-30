import {
  Anchor,
  Container,
  Stack,
  Card,
  Center,
  Loader,
  Alert,
  Image,
  Text,
} from "@mantine/core";
import {
  IconAlertCircle,
  IconCopy,
  IconLogin,
  IconPlayerPlay,
  IconRefresh,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useState, useEffect } from "react";
import { ActionButton } from "@components/buttons";
import { TextButton } from "@components/buttons";
import { config } from "@/config/env";
import { useAuth } from "@/providers/AuthProvider";
import { useCopyGameIntent } from "@/common/hooks/useCopyGameIntent";
import { GameEditModal } from "@/features/games/components/GameEditModal";
import logo from "@/assets/logos/colorful/ChatGameLab-Logo-2025-Square-Colorful2-Black-Text.png-Black-Text-Transparent.png";

interface GuestGameInfo {
  gameId: string;
  /** Only a public game may be copied; a share link alone does not release it. */
  public: boolean;
  name: string;
  description?: string;
  remaining?: number | null; // null = unlimited, 0 = exhausted
}

type ErrorType = "invalid" | "expired" | "network";

export type GuestStartMode = "new" | "continue";

const SESSION_STORAGE_KEY_PREFIX = "cgl-guest-session-";

export interface GuestCopyInfo {
  gameId: string;
  public: boolean;
}

interface GuestWelcomeProps {
  token: string;
  onStart: (mode: GuestStartMode) => void;
  /** Lets the player screen offer the same copy action while playing. */
  onInfoLoaded?: (info: GuestCopyInfo) => void;
}

export function GuestWelcome({
  token,
  onStart,
  onInfoLoaded,
}: GuestWelcomeProps) {
  const hasExistingSession = (() => {
    try {
      return !!sessionStorage.getItem(SESSION_STORAGE_KEY_PREFIX + token);
    } catch {
      return false;
    }
  })();
  const { t } = useTranslation("common");
  const { isAuthenticated } = useAuth();
  const copy = useCopyGameIntent(`/play/${token}`);
  const [gameInfo, setGameInfo] = useState<GuestGameInfo | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [errorType, setErrorType] = useState<ErrorType | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;

    async function fetchInfo() {
      try {
        const response = await fetch(
          `${config.API_BASE_URL}/play/${token}/info`,
        );
        if (!response.ok) {
          if (!cancelled) {
            setError(t("guestPlay.welcome.invalidLink"));
            setErrorType("invalid");
            setLoading(false);
          }
          return;
        }
        const data: GuestGameInfo = await response.json();
        if (!cancelled) {
          if (data.remaining === 0) {
            setError(t("guestPlay.welcome.expired"));
            setErrorType("expired");
            setLoading(false);
            return;
          }
          setGameInfo(data);
          onInfoLoaded?.({ gameId: data.gameId, public: data.public });
          setLoading(false);
        }
      } catch {
        if (!cancelled) {
          setError(t("guestPlay.welcome.loadError"));
          setErrorType("network");
          setLoading(false);
        }
      }
    }

    fetchInfo();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- fetched once per token
  }, [token, t]);

  // Back from login with a game to copy: open its dialogue once.
  useEffect(() => {
    if (gameInfo && copy.ready) copy.resume();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- consumed once
  }, [gameInfo, copy.ready]);

  if (loading) {
    return (
      <Container size="sm" py="xl">
        <Center py="xl">
          <Loader size="lg" />
        </Center>
      </Container>
    );
  }

  if (error) {
    return (
      <Container size="sm" py="xl">
        <Stack gap="xl" align="center">
          <Card shadow="sm" padding={40} radius="md" withBorder w="100%">
            <Stack align="center" gap="lg">
              <Alert
                icon={<IconAlertCircle size={16} />}
                color={errorType === "expired" ? "orange" : "red"}
                w="100%"
              >
                {error}
              </Alert>
              <Anchor href="/" size="sm">
                {t("guestPlay.welcome.backToHome")}
              </Anchor>
            </Stack>
          </Card>
          <Branding />
        </Stack>
      </Container>
    );
  }

  return (
    <Container size="sm" py="xl">
      <Stack gap="xl" align="center">
        <Card shadow="sm" padding={40} radius="md" withBorder w="100%">
          <Stack gap="lg" align="center">
            <Text size="md" c="dimmed" ta="center">
              {t("guestPlay.welcome.headline")}
            </Text>

            <Text
              size="xxl"
              fw={700}
              ta="center"
              c="gray.9"
              fz={{ base: 24, sm: 28 }}
            >
              {gameInfo?.name}
            </Text>

            {gameInfo?.description && (
              <Text size="md" c="dimmed" ta="center" maw={460}>
                {gameInfo.description}
              </Text>
            )}

            {hasExistingSession && (
              <Text size="sm" c="dimmed" ta="center" mt="xs">
                {t("guestPlay.welcome.sessionExists")}
              </Text>
            )}

            {hasExistingSession ? (
              <Stack gap="xs" w="100%" mt="md">
                <ActionButton
                  onClick={() => onStart("continue")}
                  fullWidth
                  leftSection={<IconPlayerPlay size={20} />}
                >
                  {t("guestPlay.welcome.continueGame")}
                </ActionButton>
                <TextButton
                  onClick={() => onStart("new")}
                  leftSection={<IconRefresh size={16} />}
                >
                  {t("guestPlay.welcome.restartGame")}
                </TextButton>
              </Stack>
            ) : (
              <ActionButton
                onClick={() => onStart("new")}
                fullWidth
                leftSection={<IconPlayerPlay size={20} />}
              >
                {t("guestPlay.welcome.startPlaying")}
              </ActionButton>
            )}

            {gameInfo?.public && (
              <TextButton
                onClick={() => void copy.start(gameInfo.gameId)}
                leftSection={
                  isAuthenticated ? <IconCopy size={16} /> : <IconLogin size={16} />
                }
              >
                {isAuthenticated
                  ? t("guestPlay.copyGame")
                  : t("guestPlay.copyGameLogin")}
              </TextButton>
            )}
          </Stack>
        </Card>

        <Branding />
      </Stack>

      <GameEditModal
        opened={copy.modal.opened}
        onClose={copy.modal.close}
        onCreate={copy.modal.onCreate}
        createLoading={copy.modal.createLoading}
        initialData={copy.modal.initialData}
      />
    </Container>
  );
}

function Branding() {
  const { t } = useTranslation("common");
  return (
    <Stack gap="sm" align="center" ta="center" mt="md">
      <Image
        src={logo}
        alt="ChatGameLab Logo"
        w={{ base: 160, sm: 220 }}
        h={{ base: 160, sm: 220 }}
        fit="contain"
      />
      <Text size="xs" c="dimmed" maw={400}>
        {t("home.splashDescription")}
      </Text>
    </Stack>
  );
}
