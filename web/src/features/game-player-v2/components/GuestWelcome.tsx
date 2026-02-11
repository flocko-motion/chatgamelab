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
  IconPlayerPlay,
  IconRefresh,
} from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useState, useEffect } from "react";
import { ActionButton } from "@components/buttons";
import { TextButton } from "@components/buttons";
import { config } from "@/config/env";
import logo from "@/assets/logos/colorful/ChatGameLab-Logo-2025-Square-Colorful2-Black-Text.png-Black-Text-Transparent.png";

interface GuestGameInfo {
  name: string;
  description?: string;
  remaining?: number | null; // null = unlimited, 0 = exhausted
}

type ErrorType = "invalid" | "expired" | "network";

export type GuestStartMode = "new" | "continue";

const SESSION_STORAGE_KEY_PREFIX = "cgl-guest-session-";

interface GuestWelcomeProps {
  token: string;
  onStart: (mode: GuestStartMode) => void;
}

export function GuestWelcome({ token, onStart }: GuestWelcomeProps) {
  const hasExistingSession = (() => {
    try {
      return !!sessionStorage.getItem(SESSION_STORAGE_KEY_PREFIX + token);
    } catch {
      return false;
    }
  })();
  const { t } = useTranslation("common");
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
  }, [token, t]);

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
          </Stack>
        </Card>

        <Branding />
      </Stack>
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
