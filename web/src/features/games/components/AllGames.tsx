import { useState } from "react";
import {
  Stack,
  Group,
  Card,
  Alert,
  SimpleGrid,
  Skeleton,
  Text,
  Badge,
  Tooltip,
  Box,
} from "@mantine/core";
import {
  useMediaQuery,
  useDebouncedValue,
  useDisclosure,
} from "@mantine/hooks";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import {
  IconAlertCircle,
  IconMoodEmpty,
  IconCopy,
  IconDownload,
  IconStar,
  IconStarFilled,
  IconEye,
  IconEdit,
  IconHeartFilled,
  IconTrash,
} from "@tabler/icons-react";
import { PageTitle } from "@components/typography";
import {
  SortSelector,
  type SortOption,
  FilterSegmentedControl,
  ExpandableSearch,
} from "@components/controls";
import {
  DataTable,
  DataTableEmptyState,
  type DataTableColumn,
} from "@components/DataTable";
import { DimmedLoader } from "@components/LoadingAnimation";
import { GenericIconButton } from "@components/buttons";
import { GameEditModal } from "./GameEditModal";
import { DeleteGameModal } from "./DeleteGameModal";
import { SponsorGameModal } from "./SponsorGameModal";
import { PrivateShareModal } from "./PrivateShareModal";
import { GameCard, type GameCardAction } from "./GameCard";
import { GamePlayButtons } from "./GamePlayButtons";
import {
  useGames,
  useCreateGame,
  useUpdateGame,
  useDeleteGame,
  useExportGameYaml,
  useWorkshop,
} from "@/api/hooks";
import { useAdmin } from "@/common/hooks/useAdmin";
import {
  useFavoriteState,
  useGameNavigation,
  useGameSessionState,
} from "../hooks";
import { useAuth } from "@/providers/AuthProvider";
import { parseSortValue } from "@/common/lib/sort";
import type { ObjGame } from "@/api/generated";
import { type GameFilter } from "@/features/play/types";
import type { CreateGameFormData } from "../types";
import {
  gameToFormData,
  getGameDateLabel,
  createGameWithExtraFields,
  downloadYamlFile,
} from "../lib";

function WorkshopBadges({ workshopId, mobile = false }: { workshopId: string; mobile?: boolean }) {
  const { data: workshop, isLoading } = useWorkshop(workshopId);
  if (isLoading) return <Skeleton height={14} width={120} radius="xl" />;
  if (!workshop) return null;

  if (mobile) {
    return (
      <Group gap={6} wrap="wrap">
        <Group gap={4} wrap="nowrap">
          <Text size="xs" c="gray.5">Workshop:</Text>
          <Badge size="xs" color="violet" variant="light">
            {workshop.name}
          </Badge>
        </Group>
        {workshop.institution?.name && (
          <Group gap={4} wrap="nowrap">
            <Text size="xs" c="gray.5">Orga:</Text>
            <Badge size="xs" color="gray" variant="light">
              {workshop.institution.name}
            </Badge>
          </Group>
        )}
      </Group>
    );
  }

  return (
    <Stack gap={2}>
      <Badge size="xs" color="violet" variant="light" style={{ maxWidth: 160 }}>
        {workshop.name}
      </Badge>
      {workshop.institution?.name && (
        <Badge size="xs" color="gray" variant="light" style={{ maxWidth: 160 }}>
          {workshop.institution.name}
        </Badge>
      )}
    </Stack>
  );
}

export function AllGames() {
  const { t } = useTranslation("common");
  const navigate = useNavigate();
  const isMobile = useMediaQuery("(max-width: 48em)");
  const { backendUser } = useAuth();
  const { isAdmin: isAdminUser } = useAdmin();

  const [filter, setFilter] = useState<GameFilter>("all");
  const [sortValue, setSortValue] = useState("modifiedAt-desc");
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedSearch] = useDebouncedValue(searchQuery, 300);

  const [sortField, sortDir] = parseSortValue(sortValue);

  // For favorites/sponsored filter, we fetch all games and filter client-side
  const apiFilter =
    filter === "favorites" || filter === "sponsored" ? "all" : filter;

  const {
    data: rawGames,
    isLoading,
    isFetching,
    error,
  } = useGames({
    search: debouncedSearch || undefined,
    sortBy: sortField as
      | "name"
      | "createdAt"
      | "modifiedAt"
      | "playCount"
      | "visibility"
      | "creator",
    sortDir,
    filter: apiFilter,
  });

  const { sessionsLoading, getSessionState: getGameSessionState } =
    useGameSessionState();
  const createGame = useCreateGame();
  const updateGame = useUpdateGame();
  const deleteGame = useDeleteGame();
  const exportGameYaml = useExportGameYaml();

  const [
    createModalOpened,
    { open: openCreateModal, close: closeCreateModal },
  ] = useDisclosure(false);
  const [viewModalOpened, { open: openViewModal, close: closeViewModal }] =
    useDisclosure(false);
  const [
    sponsorModalOpened,
    { open: openSponsorModal, close: closeSponsorModal },
  ] = useDisclosure(false);
  const [gameToView, setGameToView] = useState<string | null>(null);
  const [gameToViewIsOwner, setGameToViewIsOwner] = useState(false);
  const [gameToSponsor, setGameToSponsor] = useState<ObjGame | null>(null);
  const [
    privateShareModalOpened,
    { open: openPrivateShareModal, close: closePrivateShareModal },
  ] = useDisclosure(false);
  const [gameToPrivateShare, setGameToPrivateShare] = useState<ObjGame | null>(
    null,
  );
  const [deleteModalOpened, { open: openDeleteModal, close: closeDeleteModal }] =
    useDisclosure(false);
  const [gameToDelete, setGameToDelete] = useState<ObjGame | null>(null);
  const [createInitialData, setCreateInitialData] =
    useState<Partial<CreateGameFormData> | null>(null);
  const {
    favoriteGameIds,
    isFavorite,
    toggleFavorite: handleToggleFavorite,
  } = useFavoriteState();
  const {
    playGame: handlePlayGame,
    continueGame: handleContinueGame,
    restartGame: handleRestartGame,
  } = useGameNavigation();

  // Apply client-side favorites/sponsored filter
  const games =
    filter === "favorites"
      ? rawGames?.filter((game) => game.id && favoriteGameIds.has(game.id))
      : filter === "sponsored"
        ? rawGames?.filter((game) => !!game.publicSponsoredApiKeyShareId)
        : rawGames;

  const isOwner = (game: ObjGame) => {
    if (!backendUser?.id || !game.creatorId) return false;
    return game.creatorId === backendUser.id;
  };

  const handleViewGame = (game: ObjGame) => {
    if (!game.id) return;
    setGameToView(game.id);
    setGameToViewIsOwner(isOwner(game));
    openViewModal();
  };

  const handleSponsorGame = () => {
    // Find the game being viewed to pass to sponsor modal
    const game = games?.find((g) => g.id === gameToView);
    if (game) {
      setGameToSponsor(game);
      openSponsorModal();
    }
  };

  const handlePrivateShare = () => {
    const game = games?.find((g) => g.id === gameToView);
    if (game) {
      setGameToPrivateShare(game);
      openPrivateShareModal();
    }
  };

  const handleCopyGame = (game: ObjGame) => {
    if (!game.id) return;
    setCreateInitialData(gameToFormData(game));
    openCreateModal();
  };

  const handleDeleteGame = (game: ObjGame) => {
    setGameToDelete(game);
    openDeleteModal();
  };

  const handleConfirmDelete = async () => {
    if (!gameToDelete?.id) return;
    try {
      await deleteGame.mutateAsync(gameToDelete.id);
      closeDeleteModal();
      setGameToDelete(null);
    } catch {
      // Error handled by mutation
    }
  };

  const handleExport = async (game: ObjGame) => {
    if (!game.id) return;
    try {
      const yaml = await exportGameYaml.mutateAsync(game.id);
      downloadYamlFile(yaml, game.name);
    } catch {
      // Error handled by mutation
    }
  };

  const handleCloseCreateModal = () => {
    closeCreateModal();
    setCreateInitialData(null);
  };

  const handleCreateGame = async (data: CreateGameFormData) => {
    try {
      const newGame = await createGameWithExtraFields(
        data,
        createGame.mutateAsync,
        updateGame.mutateAsync,
      );
      closeCreateModal();
      setCreateInitialData(null);
      if (newGame.id) {
        navigate({ to: `/my-games/${newGame.id}` as "/" });
      }
    } catch {
      // Error handled by mutation
    }
  };

  const getDateLabel = (game: ObjGame) => getGameDateLabel(game, sortField);

  const getCardActions = (game: ObjGame): GameCardAction[] => {
    const canEdit = isOwner(game) || isAdminUser;
    const actions: GameCardAction[] = [
      canEdit
        ? {
            key: "edit",
            icon: <IconEdit size={16} color="var(--mantine-color-blue-6)" />,
            label: t("games.actions.edit"),
            onClick: () => handleViewGame(game),
          }
        : {
            key: "view",
            icon: <IconEye size={16} />,
            label: t("games.actions.view"),
            onClick: () => handleViewGame(game),
          },
    ];
    if (!isAdminUser) {
      actions.push({
        key: "copy",
        icon: <IconCopy size={16} />,
        label: t("allGames.copyGame"),
        onClick: () => handleCopyGame(game),
      });
    }
    actions.push({
      key: "export",
      icon: <IconDownload size={16} />,
      label: t("games.importExport.exportButton"),
      onClick: () => handleExport(game),
    });
    if (isAdminUser) {
      actions.push({
        key: "delete",
        icon: <IconTrash size={16} color="var(--mantine-color-red-6)" />,
        label: t("delete"),
        onClick: () => handleDeleteGame(game),
      });
    }
    return actions;
  };

  const playLabels = {
    play: t("allGames.play"),
    continue: t("allGames.continue"),
    restart: t("allGames.restart"),
  };

  const renderPlayButton = (game: ObjGame) => {
    const { hasSession, session } = getGameSessionState(game);
    return (
      <GamePlayButtons
        game={game}
        hasSession={hasSession}
        session={session}
        onPlay={handlePlayGame}
        onContinue={handleContinueGame}
        onRestart={handleRestartGame}
        labels={playLabels}
      />
    );
  };

  const columns: DataTableColumn<ObjGame>[] = [
    {
      key: "favorite",
      header: "",
      width: 40,
      render: (game) => (
        <div onClick={(e) => e.stopPropagation()}>
          <Tooltip
            label={
              isFavorite(game)
                ? t("allGames.unfavorite")
                : t("allGames.favorite")
            }
            withArrow
          >
            <GenericIconButton
              icon={
                isFavorite(game) ? (
                  <IconStarFilled
                    size={18}
                    color="var(--mantine-color-yellow-5)"
                  />
                ) : (
                  <IconStar size={18} />
                )
              }
              onClick={() => handleToggleFavorite(game)}
              aria-label={
                isFavorite(game)
                  ? t("allGames.unfavorite")
                  : t("allGames.favorite")
              }
            />
          </Tooltip>
        </div>
      ),
    },
    {
      key: "name",
      header: t("games.fields.name"),
      render: (game) => (
        <Stack gap={2}>
          <Group gap="xs" wrap="nowrap">
            <Text fw={600} size="sm" c="gray.8" lineClamp={1}>
              {game.name}
            </Text>
            {game.publicSponsoredApiKeyShareId && (
              <Tooltip
                label={t("games.sponsor.sponsoredTooltip")}
                withArrow
                multiline
                w={250}
              >
                <Badge
                  size="xs"
                  color="pink"
                  variant="light"
                  leftSection={<IconHeartFilled size={10} />}
                  style={{ flexShrink: 0, cursor: "help" }}
                >
                  {t("games.sponsor.sponsored")}
                </Badge>
              </Tooltip>
            )}
          </Group>
          {game.description && (
            <Text size="xs" c="gray.5" lineClamp={1}>
              {game.description}
            </Text>
          )}
        </Stack>
      ),
    },
    {
      key: "creator",
      header: t("games.fields.creator"),
      width: 220,
      render: (game) =>
        isOwner(game) ? (
          <Tooltip
            label={game.creatorName}
            withArrow
            disabled={!game.creatorName}
          >
            <Badge size="sm" color="accent" variant="light">
              {t("games.fields.me")}
            </Badge>
          </Tooltip>
        ) : (
          <Tooltip
            label={game.creatorName}
            withArrow
            disabled={!game.creatorName}
          >
            <Text size="sm" c="gray.6" lineClamp={1}>
              {game.creatorName || "-"}
            </Text>
          </Tooltip>
        ),
    },
    ...(isAdminUser
      ? [
          {
            key: "workshop",
            header: t("allGames.workshop"),
            width: 170,
            render: (game: ObjGame) =>
              game.workshopId ? (
                <WorkshopBadges workshopId={game.workshopId} />
              ) : (
                <Text size="sm" c="gray.4">
                  —
                </Text>
              ),
          } as DataTableColumn<ObjGame>,
        ]
      : []),
    {
      key: "playCount",
      header: t("games.fields.playCount"),
      width: 80,
      render: (game) => (
        <Text size="sm" c="gray.6" ta="center">
          {game.playCount ?? 0}
        </Text>
      ),
    },
    {
      key: "date",
      header:
        sortField === "createdAt"
          ? t("games.fields.created")
          : t("games.fields.modified"),
      width: 100,
      render: (game) => {
        const dateValue =
          sortField === "createdAt"
            ? game.meta?.createdAt
            : game.meta?.modifiedAt;
        const date = dateValue ? new Date(dateValue) : null;
        return (
          <Tooltip
            label={date ? date.toLocaleString() : "-"}
            withArrow
            disabled={!date}
          >
            <Text size="sm" c="gray.6">
              {date ? date.toLocaleDateString() : "-"}
            </Text>
          </Tooltip>
        );
      },
    },
    {
      key: "actions",
      header: "",
      width: 180,
      render: (game) => (
        <Group gap="xs" onClick={(e) => e.stopPropagation()} wrap="nowrap">
          <Box style={{ width: 140, flexShrink: 0 }}>
            {renderPlayButton(game)}
          </Box>
          <Group gap={4} wrap="nowrap">
            {isOwner(game) || isAdminUser ? (
              <Tooltip label={t("games.actions.edit")} withArrow>
                <GenericIconButton
                  icon={<IconEdit size={16} color="var(--mantine-color-blue-6)" />}
                  onClick={() => handleViewGame(game)}
                  aria-label={t("games.actions.edit")}
                />
              </Tooltip>
            ) : (
              <Tooltip label={t("games.actions.view")} withArrow>
                <GenericIconButton
                  icon={<IconEye size={16} />}
                  onClick={() => handleViewGame(game)}
                  aria-label={t("games.actions.view")}
                />
              </Tooltip>
            )}
            {!isAdminUser && (
              <Tooltip label={t("allGames.copyGame")} withArrow>
                <GenericIconButton
                  icon={<IconCopy size={16} />}
                  onClick={() => handleCopyGame(game)}
                  aria-label={t("allGames.copyGame")}
                />
              </Tooltip>
            )}
            <Tooltip label={t("games.importExport.exportButton")} withArrow>
              <GenericIconButton
                icon={<IconDownload size={16} />}
                onClick={() => handleExport(game)}
                aria-label={t("games.importExport.exportButton")}
              />
            </Tooltip>
            {isAdminUser && (
              <Tooltip label={t("delete")} withArrow>
                <GenericIconButton
                  icon={<IconTrash size={16} color="var(--mantine-color-red-6)" />}
                  onClick={() => handleDeleteGame(game)}
                  aria-label={t("delete")}
                />
              </Tooltip>
            )}
          </Group>
        </Group>
      ),
    },
  ];

  const filterOptions = [
    { value: "all", label: t("allGames.filters.all") },
    { value: "sponsored", label: t("allGames.filters.sponsored") },
    { value: "favorites", label: t("allGames.filters.favorites") },
    { value: "own", label: t("allGames.filters.own") },
    { value: "public", label: t("allGames.filters.public") },
  ];

  const sortOptions: SortOption[] = [
    { value: "modifiedAt-desc", label: t("games.sort.modifiedAt-desc") },
    { value: "modifiedAt-asc", label: t("games.sort.modifiedAt-asc") },
    { value: "createdAt-desc", label: t("games.sort.createdAt-desc") },
    { value: "createdAt-asc", label: t("games.sort.createdAt-asc") },
    { value: "name-asc", label: t("games.sort.name-asc") },
    { value: "name-desc", label: t("games.sort.name-desc") },
    { value: "playCount-desc", label: t("games.sort.playCount-desc") },
    { value: "playCount-asc", label: t("games.sort.playCount-asc") },
  ];

  const hasData = rawGames !== undefined;
  const isInitialLoading = !hasData && (isLoading || sessionsLoading);
  const isRefetching = isFetching && hasData;

  if (isInitialLoading) {
    return (
      <Stack gap="xl">
        <Skeleton height={40} width="50%" />
        <Skeleton height={36} width={300} />
        <Skeleton height={300} />
      </Stack>
    );
  }

  if (error) {
    return (
      <Alert
        icon={<IconAlertCircle size={16} />}
        title={t("errors.titles.error")}
        color="red"
      >
        {t("games.errors.loadFailed")}
      </Alert>
    );
  }

  return (
    <>
      <Stack
        gap="lg"
        h={{ base: "calc(100vh - 180px)", sm: "calc(100vh - 280px)" }}
        style={{ overflow: "hidden" }}
      >
        {/* Sticky header section */}
        <Stack gap="lg" style={{ flexShrink: 0 }}>
          <PageTitle>{t("allGames.title")}</PageTitle>

          {isMobile ? (
            <Group gap="sm" wrap="nowrap">
              <ExpandableSearch
                value={searchQuery}
                onChange={setSearchQuery}
                placeholder={t("search")}
              />
              <Group gap="xs" wrap="nowrap" style={{ flexShrink: 0 }}>
                <FilterSegmentedControl
                  value={filter}
                  onChange={setFilter}
                  options={filterOptions}
                />
                <SortSelector
                  options={sortOptions}
                  value={sortValue}
                  onChange={setSortValue}
                  label={t("games.sort.label")}
                />
              </Group>
            </Group>
          ) : (
            <Group justify="flex-end" wrap="wrap" gap="sm">
              <Group gap="sm" wrap="wrap">
                <ExpandableSearch
                  value={searchQuery}
                  onChange={setSearchQuery}
                  placeholder={t("search")}
                />
                <FilterSegmentedControl
                  value={filter}
                  onChange={setFilter}
                  options={filterOptions}
                />
                <SortSelector
                  options={sortOptions}
                  value={sortValue}
                  onChange={setSortValue}
                  label={t("games.sort.label")}
                />
              </Group>
            </Group>
          )}
        </Stack>

        {/* Scrollable content area */}
        <Box style={{ flex: 1, minHeight: 0, overflow: "auto" }}>
          <DimmedLoader visible={isRefetching} loaderSize="lg">
            {isMobile ? (
              (games?.length ?? 0) === 0 ? (
                <Card shadow="sm" p="xl" radius="md" withBorder>
                  <Stack align="center" gap="md" py="xl">
                    <IconMoodEmpty
                      size={48}
                      color="var(--mantine-color-gray-5)"
                    />
                    <Text c="gray.6" ta="center">
                      {t("allGames.empty.title")}
                    </Text>
                    <Text size="sm" c="gray.5" ta="center">
                      {t("allGames.empty.description")}
                    </Text>
                  </Stack>
                </Card>
              ) : (
                <SimpleGrid cols={1} spacing="md">
                  {games?.map((game) => {
                    const { hasSession, session } = getGameSessionState(game);
                    return (
                      <GameCard
                        key={game.id}
                        game={game}
                        onPlay={() => handlePlayGame(game)}
                        playLabel={t("allGames.play")}
                        hasSession={hasSession}
                        onContinue={
                          session
                            ? () => handleContinueGame(session)
                            : undefined
                        }
                        continueLabel={t("allGames.continue")}
                        onRestart={
                          session
                            ? () => handleRestartGame(game, session)
                            : undefined
                        }
                        restartLabel={t("allGames.restart")}
                        showCreator
                        isOwner={isOwner(game)}
                        creatorLabel={t("games.fields.me")}
                        isFavorite={isFavorite(game)}
                        onToggleFavorite={() => handleToggleFavorite(game)}
                        favoriteLabel={t("allGames.favorite")}
                        unfavoriteLabel={t("allGames.unfavorite")}
                        actions={getCardActions(game)}
                        dateLabel={getDateLabel(game)}
                        extra={
                          isAdminUser && game.workshopId ? (
                            <WorkshopBadges workshopId={game.workshopId} mobile />
                          ) : undefined
                        }
                      />
                    );
                  })}
                </SimpleGrid>
              )
            ) : (
              <DataTable
                data={games ?? []}
                columns={columns}
                getRowKey={(game) => game.id || ""}
                onRowClick={handleViewGame}
                isLoading={false}
                fillHeight
                getRowStyle={(game) =>
                  game.publicSponsoredApiKeyShareId
                    ? {
                        borderLeft: "3px solid var(--mantine-color-pink-4)",
                      }
                    : undefined
                }
                renderMobileCard={(game) => {
                  const { hasSession, session } = getGameSessionState(game);
                  return (
                    <GameCard
                      game={game}
                      onPlay={() => handlePlayGame(game)}
                      playLabel={t("allGames.play")}
                      hasSession={hasSession}
                      onContinue={
                        session ? () => handleContinueGame(session) : undefined
                      }
                      continueLabel={t("allGames.continue")}
                      onRestart={
                        session
                          ? () => handleRestartGame(game, session)
                          : undefined
                      }
                      restartLabel={t("allGames.restart")}
                      showCreator
                      isOwner={isOwner(game)}
                      creatorLabel={t("games.fields.me")}
                      isFavorite={isFavorite(game)}
                      onToggleFavorite={() => handleToggleFavorite(game)}
                      favoriteLabel={t("allGames.favorite")}
                      unfavoriteLabel={t("allGames.unfavorite")}
                      actions={getCardActions(game)}
                      dateLabel={getDateLabel(game)}
                    />
                  );
                }}
                emptyState={
                  <DataTableEmptyState
                    icon={
                      <IconMoodEmpty
                        size={48}
                        color="var(--mantine-color-gray-5)"
                      />
                    }
                    title={t("allGames.empty.title")}
                    description={t("allGames.empty.description")}
                  />
                }
              />
            )}
          </DimmedLoader>
        </Box>
      </Stack>

      <GameEditModal
        opened={createModalOpened}
        onClose={handleCloseCreateModal}
        onCreate={handleCreateGame}
        createLoading={createGame.isPending}
        initialData={createInitialData}
      />

      <GameEditModal
        gameId={gameToView}
        opened={viewModalOpened}
        onClose={() => {
          closeViewModal();
          setGameToView(null);
        }}
        readOnly={!gameToViewIsOwner && !isAdminUser}
        isOwner={gameToViewIsOwner}
        onSponsor={gameToViewIsOwner ? handleSponsorGame : undefined}
        onPrivateShare={gameToViewIsOwner ? handlePrivateShare : undefined}
        onCopy={
          !gameToViewIsOwner && !isAdminUser
            ? () => {
                const game = games?.find((g) => g.id === gameToView);
                if (game) {
                  closeViewModal();
                  setGameToView(null);
                  handleCopyGame(game);
                }
              }
            : undefined
        }
      />

      <DeleteGameModal
        opened={deleteModalOpened}
        onClose={() => {
          closeDeleteModal();
          setGameToDelete(null);
        }}
        onConfirm={handleConfirmDelete}
        gameName={gameToDelete?.name ?? ""}
        loading={deleteGame.isPending}
        isOwner={gameToDelete ? isOwner(gameToDelete) : false}
      />

      <SponsorGameModal
        game={gameToSponsor}
        opened={sponsorModalOpened}
        onClose={() => {
          closeSponsorModal();
          setGameToSponsor(null);
        }}
      />

      <PrivateShareModal
        game={gameToPrivateShare}
        opened={privateShareModalOpened}
        onClose={() => {
          closePrivateShareModal();
          setGameToPrivateShare(null);
        }}
      />
    </>
  );
}
