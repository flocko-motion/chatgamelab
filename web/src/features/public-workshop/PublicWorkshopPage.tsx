import { useEffect, useState } from "react";
import {
  Alert,
  Button,
  Card,
  Center,
  Container,
  Group,
  Loader,
  SimpleGrid,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import { useDisclosure } from "@mantine/hooks";
import {
  IconCopy,
  IconDownload,
  IconLogin,
  IconPlayerPlay,
  IconWorldOff,
} from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { apiClient } from "@/api/client";
import { config } from "@/config/env";
import { useAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { useCreateGame, useUpdateGame } from "@/api/hooks";
import type { ObjPublicWorkshopGame } from "@/api/generated";
import { useAuth } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { publicWorkshopPath } from "@/common/lib/publicWorkshop";
import { rememberReturnTo, takeReturnTo } from "@/common/lib/returnTo";
import { GameEditModal } from "@/features/games/components/GameEditModal";
import {
  createGameWithExtraFields,
  downloadYamlFile,
  gameToFormData,
} from "@/features/games/lib";
import type { CreateGameFormData } from "@/features/games/types";

interface PublicWorkshopPageProps {
  slug: string;
}

/**
 * The page a workshop shows the world at /w/<slug>. It never names who made a
 * game; the server leaves creators out of the response.
 */
export function PublicWorkshopPage({ slug }: PublicWorkshopPageProps) {
  const { t } = useTranslation("common");
  const navigate = useNavigate();
  const { isAuthenticated, isParticipant } = useAuth();
  const authApi = useAuthenticatedApi();
  const createGame = useCreateGame();
  const updateGame = useUpdateGame();
  const [copyData, setCopyData] = useState<Partial<CreateGameFormData> | null>(
    null,
  );
  const [copyingId, setCopyingId] = useState<string | null>(null);
  const [copyModalOpened, { open: openCopyModal, close: closeCopyModal }] =
    useDisclosure(false);

  // Back from login or registration: the visitor has arrived.
  useEffect(() => {
    if (isAuthenticated) takeReturnTo();
  }, [isAuthenticated]);

  const { data: page, isLoading, isError } = useQuery({
    queryKey: ["publicWorkshop", slug],
    queryFn: async () => (await apiClient.public.workshopsDetail(slug)).data,
    retry: false,
  });

  const handleDownload = async (game: ObjPublicWorkshopGame) => {
    if (!game.id) return;
    const response = await fetch(
      `${config.API_BASE_URL}/public/workshops/${encodeURIComponent(slug)}/games/${game.id}/yaml`,
    );
    if (response.ok) {
      downloadYamlFile(await response.text(), game.name);
    }
  };

  const handleCopy = async (game: ObjPublicWorkshopGame) => {
    if (!game.id) return;
    if (!isAuthenticated || !authApi) {
      rememberReturnTo(publicWorkshopPath(slug));
      navigate({ to: ROUTES.AUTH_LOGIN });
      return;
    }
    setCopyingId(game.id);
    try {
      const full = (await authApi.games.gamesDetail(game.id)).data;
      setCopyData(gameToFormData(full));
      openCopyModal();
    } finally {
      setCopyingId(null);
    }
  };

  const handleCreateCopy = async (data: CreateGameFormData) => {
    try {
      const newGame = await createGameWithExtraFields(
        data,
        createGame.mutateAsync,
        updateGame.mutateAsync,
      );
      closeCopyModal();
      setCopyData(null);
      if (isParticipant) {
        navigate({ to: ROUTES.MY_WORKSHOP as "/" });
      } else if (newGame.id) {
        navigate({ to: `/my-games/${newGame.id}` as "/" });
      }
    } catch {
      // Error handled by mutation
    }
  };

  if (isLoading) {
    return (
      <Center h="50vh">
        <Loader size="lg" />
      </Center>
    );
  }

  if (isError || !page) {
    return (
      <Container size="xs" py="xl">
        <Card shadow="sm" padding="xl" radius="md" withBorder>
          <Stack gap="md" align="center" ta="center">
            <IconWorldOff size={40} color="var(--mantine-color-gray-5)" />
            <Title order={3}>{t("publicWorkshop.unavailable.title")}</Title>
            <Text size="sm" c="dimmed">
              {t("publicWorkshop.unavailable.message")}
            </Text>
            <Button component={Link} to={ROUTES.HOME} variant="light">
              {t("publicWorkshop.unavailable.home")}
            </Button>
          </Stack>
        </Card>
      </Container>
    );
  }

  const games = page.games ?? [];

  return (
    <Container size="md" py="xl">
      <Stack gap="xl">
        <Stack gap="xs">
          <Title order={1}>{page.name}</Title>
          {page.description && (
            <Text style={{ whiteSpace: "pre-line" }}>{page.description}</Text>
          )}
        </Stack>

        <Stack gap="sm">
          <Title order={3}>{t("publicWorkshop.gamesTitle")}</Title>
          {games.length === 0 ? (
            <Alert color="gray" variant="light">
              {t("publicWorkshop.noGames")}
            </Alert>
          ) : (
            <SimpleGrid cols={{ base: 1, sm: 2 }} spacing="md">
              {games.map((game) => (
                <PublicGameCard
                  key={game.id}
                  game={game}
                  isAuthenticated={isAuthenticated}
                  copying={copyingId === game.id}
                  onDownload={() => handleDownload(game)}
                  onCopy={() => handleCopy(game)}
                />
              ))}
            </SimpleGrid>
          )}
        </Stack>
      </Stack>

      <GameEditModal
        opened={copyModalOpened}
        onClose={() => {
          closeCopyModal();
          setCopyData(null);
        }}
        onCreate={handleCreateCopy}
        createLoading={createGame.isPending}
        initialData={copyData}
      />
    </Container>
  );
}

interface PublicGameCardProps {
  game: ObjPublicWorkshopGame;
  isAuthenticated: boolean;
  copying: boolean;
  onDownload: () => void;
  onCopy: () => void;
}

function PublicGameCard({
  game,
  isAuthenticated,
  copying,
  onDownload,
  onCopy,
}: PublicGameCardProps) {
  const { t } = useTranslation("common");
  const play = game.play;
  const exhausted = !!play && (play.remaining ?? 0) <= 0;

  return (
    <Card shadow="sm" padding="lg" radius="md" withBorder>
      <Stack gap="sm" h="100%">
        <Title order={4}>{game.name}</Title>
        {game.description && (
          <Text size="sm" c="dimmed" style={{ whiteSpace: "pre-line" }}>
            {game.description}
          </Text>
        )}
        <Stack gap={4} mt="auto">
          {play && (
            <>
              {exhausted ? (
                <Button leftSection={<IconPlayerPlay size={16} />} disabled>
                  {t("publicWorkshop.play")}
                </Button>
              ) : (
                <Button
                  component={Link}
                  to={`/play/${play.token}` as "/"}
                  leftSection={<IconPlayerPlay size={16} />}
                >
                  {t("publicWorkshop.play")}
                </Button>
              )}
              <Text size="xs" c="dimmed" ta="center">
                {exhausted
                  ? t("publicWorkshop.playExhausted")
                  : t("publicWorkshop.playRemaining", {
                      remaining: play.remaining,
                      limit: play.limit,
                    })}
              </Text>
            </>
          )}
          <Group grow gap="xs">
            <Button
              variant="light"
              leftSection={<IconDownload size={16} />}
              onClick={onDownload}
            >
              {t("publicWorkshop.download")}
            </Button>
            <Button
              variant="light"
              leftSection={
                isAuthenticated ? (
                  <IconCopy size={16} />
                ) : (
                  <IconLogin size={16} />
                )
              }
              onClick={onCopy}
              loading={copying}
            >
              {isAuthenticated
                ? t("publicWorkshop.copy")
                : t("publicWorkshop.copyLogin")}
            </Button>
          </Group>
        </Stack>
      </Stack>
    </Card>
  );
}
