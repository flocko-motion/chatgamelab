import { useEffect } from "react";
import {
  Alert,
  Anchor,
  Button,
  Card,
  Center,
  Container,
  Flex,
  Loader,
  SimpleGrid,
  Spoiler,
  Stack,
  Text,
  Title,
} from "@mantine/core";
import {
  IconCopy,
  IconDownload,
  IconLogin,
  IconPlayerPlay,
  IconSparkles,
  IconWorldOff,
} from "@tabler/icons-react";
import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { apiClient } from "@/api/client";
import { config } from "@/config/env";
import type {
  ObjPublicWorkshopGame,
  ObjPublicWorkshopLink,
} from "@/api/generated";
import { useAuth } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { useCopyGameIntent } from "@/common/hooks/useCopyGameIntent";
import { EXTERNAL_LINKS } from "@/config/externalLinks";
import { publicWorkshopPath } from "@/common/lib/publicWorkshop";
import { buildShareUrl } from "@/common/lib/url";
import { takeReturnTo } from "@/common/lib/returnTo";
import { GameEditModal } from "@/features/games/components/GameEditModal";
import { downloadYamlFile } from "@/features/games/lib";

interface PublicWorkshopPageProps {
  slug: string;
  /** From ?copy=<id>: the "Kopieren" link opens this page in a new tab. */
  copyGameId?: string;
}

/**
 * The page a workshop shows the world at /w/<slug>. It never names who made a
 * game; the server leaves creators out of the response.
 */
export function PublicWorkshopPage({
  slug,
  copyGameId,
}: PublicWorkshopPageProps) {
  const { t } = useTranslation("common");
  const navigate = useNavigate();
  const { isAuthenticated, backendUser } = useAuth();
  const copy = useCopyGameIntent(publicWorkshopPath(slug));

  // Back from login or registration: the visitor has arrived. Waiting for
  // backendUser matters — a new account is authenticated at Auth0 before the
  // registration form appears, and that form still needs the remembered path.
  useEffect(() => {
    if (backendUser) takeReturnTo();
  }, [backendUser]);

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

  // Back from login or registration with a game to copy: open its dialogue once.
  useEffect(() => {
    if (page && copy.ready) copy.resume();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- consumed once
  }, [page, copy.ready, slug]);

  // Arrived through the "Kopieren" link in a fresh tab. The parameter is
  // dropped from the address first, so a reload does not repeat the dialogue.
  useEffect(() => {
    if (!copyGameId || !page || !copy.ready) return;
    void navigate({ to: ".", search: {}, replace: true });
    void copy.start(copyGameId);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- consumed once
  }, [copyGameId, page, copy.ready]);

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
          <AboutChatGameLab />
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
                  playAvailable={page.playAvailable ?? false}
                  copyUrl={buildShareUrl(
                    `${publicWorkshopPath(slug)}?copy=${game.id}`,
                  )}
                  onDownload={() => handleDownload(game)}
                />
              ))}
            </SimpleGrid>
          )}
        </Stack>

        <FurtherReading links={page.links ?? []} />

        <CallToAction />
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

/** One line on what ChatGameLab is, for visitors who have never heard of it. */
function AboutChatGameLab() {
  const { t } = useTranslation("common");

  return (
    <Stack gap={2}>
      <Text size="sm" c="dimmed">
        {t("publicWorkshop.about.oneLiner")}
      </Text>
      <Spoiler
        maxHeight={0}
        showLabel={t("publicWorkshop.about.more")}
        hideLabel={t("publicWorkshop.about.less")}
        styles={{ control: { fontSize: "var(--mantine-font-size-sm)" } }}
      >
        <Text size="sm" c="dimmed" mt="xs">
          {t("publicWorkshop.about.details")}{" "}
          <Anchor
            href={EXTERNAL_LINKS.CHATGAMELAB.href}
            target="_blank"
            rel="noopener noreferrer"
            size="sm"
          >
            {t("publicWorkshop.about.link")}
          </Anchor>
        </Text>
      </Spoiler>
    </Stack>
  );
}

/** Links the workshop's leaders added, shown below the games. */
function FurtherReading({ links }: { links: ObjPublicWorkshopLink[] }) {
  const { t } = useTranslation("common");

  if (links.length === 0) return null;

  return (
    <Stack gap="sm">
      <Title order={3}>{t("publicWorkshop.links.title")}</Title>
      <Stack gap="xs">
        {links.map((link) => (
          <Card key={link.url} withBorder radius="md" padding="md">
            <Anchor
              href={link.url}
              target="_blank"
              rel="noopener noreferrer"
              fw={500}
            >
              {link.title}
            </Anchor>
            {link.description && (
              <Text size="sm" c="dimmed">
                {link.description}
              </Text>
            )}
          </Card>
        ))}
      </Stack>
    </Stack>
  );
}

/** Closing invitation to build a game of one's own. */
function CallToAction() {
  const { t } = useTranslation("common");

  return (
    <Card withBorder radius="md" padding="lg">
      <Stack gap="xs" align="center" ta="center">
        <IconSparkles size={28} />
        <Title order={3}>{t("publicWorkshop.cta.title")}</Title>
        <Text size="sm" c="dimmed">
          {t("publicWorkshop.cta.text")}
        </Text>
        <Button component={Link} to={ROUTES.AUTH_REGISTER} mt="xs">
          {t("publicWorkshop.cta.button")}
        </Button>
      </Stack>
    </Card>
  );
}

interface PublicGameCardProps {
  game: ObjPublicWorkshopGame;
  isAuthenticated: boolean;
  /** False when the workshop has no key: no game here can be played. */
  playAvailable: boolean;
  /** This page with ?copy=<id>, opened in a new tab. */
  copyUrl: string;
  onDownload: () => void;
}

function PublicGameCard({
  game,
  isAuthenticated,
  playAvailable,
  copyUrl,
  onDownload,
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
                  component="a"
                  href={buildShareUrl(`/play/${play.token}`)}
                  target="_blank"
                  rel="noopener noreferrer"
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
          {!playAvailable && (
            <>
              <Button leftSection={<IconPlayerPlay size={16} />} disabled>
                {t("publicWorkshop.play")}
              </Button>
              <Text size="xs" c="dimmed" ta="center">
                {t("publicWorkshop.playNoKey")}
              </Text>
            </>
          )}
          {/* Side by side while both fit, else one full-width row each. */}
          <Flex gap="xs" wrap="wrap">
            <Button
              variant="light"
              leftSection={<IconDownload size={16} />}
              onClick={onDownload}
              style={{ flex: "1 1 auto" }}
            >
              {t("publicWorkshop.download")}
            </Button>
            <Button
              variant="light"
              component="a"
              href={copyUrl}
              target="_blank"
              rel="noopener noreferrer"
              leftSection={
                isAuthenticated ? (
                  <IconCopy size={16} />
                ) : (
                  <IconLogin size={16} />
                )
              }
              style={{ flex: "1 1 auto" }}
            >
              {isAuthenticated
                ? t("publicWorkshop.copy")
                : t("publicWorkshop.copyLogin")}
            </Button>
          </Flex>
        </Stack>
      </Stack>
    </Card>
  );
}
