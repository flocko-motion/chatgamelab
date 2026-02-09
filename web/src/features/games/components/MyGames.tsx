import React, { useState } from "react";
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
  useDisclosure,
  useMediaQuery,
  useDebouncedValue,
} from "@mantine/hooks";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import {
  IconAlertCircle,
  IconMoodEmpty,
  IconCopy,
  IconDownload,
  IconWorld,
  IconLock,
  IconStar,
  IconStarFilled,
  IconFileImport,
} from "@tabler/icons-react";
import {
  ActionButton,
  TextButton,
  PlayGameButton,
  EditIconButton,
  DeleteIconButton,
  GenericIconButton,
  PlusIconButton,
} from "@components/buttons";
import {
  SortSelector,
  type SortOption,
  FilterSegmentedControl,
  ExpandableSearch,
} from "@components/controls";
import { PageTitle } from "@components/typography";
import {
  DataTable,
  DataTableEmptyState,
  type DataTableColumn,
} from "@components/DataTable";
import { DimmedLoader } from "@components/LoadingAnimation";
import {
  useGames,
  useCreateGame,
  useUpdateGame,
  useDeleteGame,
  useExportGameYaml,
  useGameSessionMap,
  useDeleteSession,
  useFavoriteGames,
  useAddFavorite,
  useRemoveFavorite,
} from "@/api/hooks";
import type { ObjGame, DbUserSessionWithGame } from "@/api/generated";
import { type CreateGameFormData } from "../types";
import { parseGameYaml, gameToFormData } from "../lib";
import { GameEditModal } from "./GameEditModal";
import { SponsorGameModal } from "./SponsorGameModal";
import { DeleteGameModal } from "./DeleteGameModal";
import { GameCard, type GameCardAction } from "./GameCard";
import { useModals } from "@mantine/modals";

interface MyGamesProps {
  initialGameId?: string;
  initialMode?: "create" | "view";
  onModalClose?: () => void;
  /** Auto-trigger the import file dialog on mount */
  autoImport?: boolean;
}

export function MyGames({
  initialGameId,
  initialMode,
  onModalClose,
  autoImport,
}: MyGamesProps = {}) {
  const { t } = useTranslation("common");
  const isMobile = useMediaQuery("(max-width: 48em)");
  const navigate = useNavigate();
  const modals = useModals();

  const [
    createModalOpened,
    { open: openCreateModal, close: closeCreateModal },
  ] = useDisclosure(initialMode === "create");
  const [
    deleteModalOpened,
    { open: openDeleteModal, close: closeDeleteModal },
  ] = useDisclosure(false);
  const [viewModalOpened, { open: openViewModal, close: closeViewModal }] =
    useDisclosure(initialGameId ? true : false);
  const [gameToDelete, setGameToDelete] = useState<ObjGame | null>(null);
  const [gameToView, setGameToView] = useState<string | null>(
    initialGameId ?? null,
  );
  const [
    sponsorModalOpened,
    { open: openSponsorModal, close: closeSponsorModal },
  ] = useDisclosure(false);
  const [gameToSponsor, setGameToSponsor] = useState<ObjGame | null>(null);
  const [sortValue, setSortValue] = useState("modifiedAt-desc");
  const [showFavorites, setShowFavorites] = useState<"all" | "favorites">(
    "all",
  );
  const [searchQuery, setSearchQuery] = useState("");
  const [debouncedSearch] = useDebouncedValue(searchQuery, 300);

  // Parse combined sort value into field and direction
  const [sortField, sortDir] = sortValue.split("-") as [string, "asc" | "desc"];

  const {
    data: rawGames,
    isLoading,
    isFetching,
    error,
  } = useGames({
    sortBy: sortField as
      | "name"
      | "createdAt"
      | "modifiedAt"
      | "playCount"
      | "visibility"
      | "creator",
    sortDir,
    filter: "own",
    search: debouncedSearch || undefined,
  });
  const { sessionMap, isLoading: sessionsLoading } = useGameSessionMap();
  const createGame = useCreateGame();
  const updateGame = useUpdateGame();
  const deleteGame = useDeleteGame();
  const deleteSession = useDeleteSession();
  const exportGameYaml = useExportGameYaml();
  const { data: favoriteGames } = useFavoriteGames();
  const addFavorite = useAddFavorite();
  const removeFavorite = useRemoveFavorite();

  const favoriteGameIds = new Set(favoriteGames?.map((g) => g.id) ?? []);

  // Apply client-side favorites filter
  const games =
    showFavorites === "favorites"
      ? rawGames?.filter((game) => game.id && favoriteGameIds.has(game.id))
      : rawGames;

  const isFavorite = (game: ObjGame) =>
    game.id ? favoriteGameIds.has(game.id) : false;

  const handleToggleFavorite = (game: ObjGame) => {
    if (!game.id) return;
    if (isFavorite(game)) {
      removeFavorite.mutate(game.id);
    } else {
      addFavorite.mutate(game.id);
    }
  };

  const fileInputRef = React.useRef<HTMLInputElement>(null);

  const handleCreateGame = async (data: CreateGameFormData) => {
    try {
      const newGame = await createGame.mutateAsync({
        name: data.name,
        description: data.description,
        public: data.isPublic,
      });

      // Update with additional fields if provided
      const hasExtraFields =
        data.systemMessageScenario ||
        data.systemMessageGameStart ||
        data.imageStyle ||
        data.statusFields;
      if (newGame.id && hasExtraFields) {
        await updateGame.mutateAsync({
          id: newGame.id,
          game: {
            ...newGame,
            systemMessageScenario: data.systemMessageScenario,
            systemMessageGameStart: data.systemMessageGameStart,
            imageStyle: data.imageStyle,
            statusFields: data.statusFields,
          },
        });
      }

      closeCreateModal();
    } catch {
      // Error handled by mutation
    }
  };

  const handleCloseCreateModal = () => {
    closeCreateModal();
    setCreateInitialData(null);
    if (initialMode === "create") {
      onModalClose?.();
    }
  };

  const handleEditGame = (game: ObjGame) => {
    if (game.id) {
      setGameToView(game.id);
      openViewModal();
    }
  };

  const handleViewGame = (game: ObjGame) => {
    if (game.id) {
      setGameToView(game.id);
      openViewModal();
    }
  };

  const handleDeleteClick = (game: ObjGame) => {
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
      const blob = new Blob([yaml], { type: "application/x-yaml" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `${game.name || "game"}.yaml`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    } catch {
      // Error handled by mutation
    }
  };

  const handleImportClick = () => {
    fileInputRef.current?.click();
  };

  // Auto-trigger import file dialog when navigated with ?action=import
  React.useEffect(() => {
    if (autoImport) {
      // Small delay to ensure the file input is mounted
      const timer = setTimeout(() => fileInputRef.current?.click(), 100);
      return () => clearTimeout(timer);
    }
  }, [autoImport]);

  // Pre-populated data for create modal (from YAML import or game copy)
  const [createInitialData, setCreateInitialData] =
    useState<Partial<CreateGameFormData> | null>(null);

  const handleCopyGame = (game: ObjGame) => {
    if (!game.id) return;
    setCreateInitialData(gameToFormData(game));
    openCreateModal();
  };

  const handlePlayGame = (game: ObjGame) => {
    if (game.id) {
      navigate({ to: "/games/$gameId/play", params: { gameId: game.id } });
    }
  };

  const handleContinueGame = (session: DbUserSessionWithGame) => {
    if (session.id) {
      navigate({ to: `/sessions/${session.id}` as "/" });
    }
  };

  const handleRestartGame = (game: ObjGame, session: DbUserSessionWithGame) => {
    if (!game.id || !session.id) return;

    modals.openConfirmModal({
      title: t("myGames.restartConfirm.title"),
      children: (
        <Text size="sm">
          {t("myGames.restartConfirm.message", {
            game: game.name || t("sessions.untitledGame"),
          })}
        </Text>
      ),
      labels: {
        confirm: t("myGames.restartConfirm.confirm"),
        cancel: t("cancel"),
      },
      confirmProps: { color: "red" },
      onConfirm: async () => {
        // Delete session - if it fails (e.g., already deleted), just continue to play
        try {
          await deleteSession.mutateAsync(session.id!);
        } catch {
          // Session may have been deleted already, ignore and continue
        }
        navigate({ to: "/games/$gameId/play", params: { gameId: game.id! } });
      },
    });
  };

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      const content = e.target?.result as string;
      const formData = parseGameYaml(content);
      setCreateInitialData(formData);
      openCreateModal();
    };
    reader.readAsText(file);
    event.target.value = "";
  };

  const getGameSessionState = (game: ObjGame) => {
    if (!game.id) return { hasSession: false, session: undefined };
    const session = sessionMap.get(game.id);
    return { hasSession: !!session, session };
  };

  const getDateLabel = (game: ObjGame) => {
    const dateValue =
      sortField === "createdAt" ? game.meta?.createdAt : game.meta?.modifiedAt;
    return dateValue ? new Date(dateValue).toLocaleDateString() : undefined;
  };

  const getCardActions = (game: ObjGame): GameCardAction[] => [
    {
      key: "edit",
      icon: null,
      label: t("editGame"),
      onClick: () => handleEditGame(game),
    },
    {
      key: "copy",
      icon: <IconCopy size={16} />,
      label: t("copyGame"),
      onClick: () => handleCopyGame(game),
    },
    {
      key: "export",
      icon: <IconDownload size={16} />,
      label: t("games.importExport.exportButton"),
      onClick: () => handleExport(game),
    },
    {
      key: "delete",
      icon: null,
      label: t("deleteGame"),
      onClick: () => handleDeleteClick(game),
    },
  ];

  const renderPlayButton = (game: ObjGame) => {
    const { hasSession, session } = getGameSessionState(game);

    if (!hasSession) {
      return (
        <PlayGameButton
          onClick={() => handlePlayGame(game)}
          size="xs"
          style={{ width: "100%" }}
        >
          {t("myGames.play")}
        </PlayGameButton>
      );
    }

    return (
      <Stack gap={4}>
        <PlayGameButton
          onClick={() => handleContinueGame(session!)}
          size="xs"
          style={{ width: "100%" }}
        >
          {t("myGames.continue")}
        </PlayGameButton>
        <TextButton onClick={() => handleRestartGame(game, session!)} size="xs">
          {t("myGames.restart")}
        </TextButton>
      </Stack>
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
              isFavorite(game) ? t("myGames.unfavorite") : t("myGames.favorite")
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
                  ? t("myGames.unfavorite")
                  : t("myGames.favorite")
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
      key: "playCount",
      header: t("games.fields.playCount"),
      width: 80,
      render: (game) => (
        <Tooltip label={t("games.fields.playCount")} withArrow>
          <Text size="sm" c="gray.6" ta="center">
            {game.playCount ?? 0}
          </Text>
        </Tooltip>
      ),
    },
    {
      key: "visibility",
      header: t("games.fields.visibility"),
      width: 150,
      render: (game) =>
        game.public ? (
          <Badge
            size="sm"
            color="green"
            variant="light"
            leftSection={<IconWorld size={12} />}
            style={{ whiteSpace: "nowrap" }}
          >
            {t("games.visibility.public")}
          </Badge>
        ) : (
          <Badge
            size="sm"
            color="gray"
            variant="light"
            leftSection={<IconLock size={12} />}
            style={{ whiteSpace: "nowrap" }}
          >
            {t("games.visibility.private")}
          </Badge>
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
      header: t("actions"),
      width: 260,
      render: (game) => (
        <Group gap="xs" onClick={(e) => e.stopPropagation()} wrap="nowrap">
          <Box style={{ width: 140, flexShrink: 0 }}>
            {renderPlayButton(game)}
          </Box>
          <Group gap={4} wrap="wrap">
            <Tooltip label={t("editGame")} withArrow>
              <EditIconButton
                onClick={() => handleEditGame(game)}
                aria-label={t("edit")}
              />
            </Tooltip>
            <Tooltip label={t("copyGame")} withArrow>
              <GenericIconButton
                icon={<IconCopy size={16} />}
                onClick={() => handleCopyGame(game)}
                aria-label={t("myGames.copyGame")}
              />
            </Tooltip>
            <Tooltip label={t("games.importExport.exportButton")} withArrow>
              <GenericIconButton
                icon={<IconDownload size={16} />}
                onClick={() => handleExport(game)}
                aria-label={t("games.importExport.exportButton")}
              />
            </Tooltip>
            <Tooltip label={t("deleteGame")} withArrow>
              <DeleteIconButton
                onClick={() => handleDeleteClick(game)}
                aria-label={t("delete")}
              />
            </Tooltip>
          </Group>
        </Group>
      ),
    },
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
    { value: "visibility-desc", label: t("games.sort.visibility-desc") },
    { value: "visibility-asc", label: t("games.sort.visibility-asc") },
  ];

  const hasData = rawGames !== undefined;
  const isInitialLoading = !hasData && (isLoading || sessionsLoading);
  const isRefetching = isFetching && hasData;

  if (isInitialLoading) {
    return (
      <Stack gap="xl">
        <Skeleton height={40} width="50%" />
        <Skeleton height={36} width={180} />
        {isMobile ? (
          <Stack gap="md">
            {[1, 2, 3].map((i) => (
              <Card key={i} shadow="sm" p="lg" radius="md" withBorder>
                <Stack gap="sm">
                  <Skeleton height={24} width="70%" />
                  <Skeleton height={16} width="90%" />
                  <Group gap="xl">
                    <Skeleton height={32} width={80} />
                    <Skeleton height={32} width={80} />
                  </Group>
                </Stack>
              </Card>
            ))}
          </Stack>
        ) : (
          <Skeleton height={300} />
        )}
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
          <PageTitle>{t("myGames.title")}</PageTitle>

          <input
            type="file"
            ref={fileInputRef}
            onChange={handleFileSelect}
            accept=".yaml,.yml"
            style={{ display: "none" }}
          />

          {isMobile ? (
            <Group gap="sm" wrap="nowrap">
              <Tooltip label={t("games.createButton")} withArrow>
                <PlusIconButton
                  onClick={openCreateModal}
                  variant="filled"
                  aria-label={t("games.createButton")}
                />
              </Tooltip>
              <Tooltip label={t("games.importExport.importButton")} withArrow>
                <GenericIconButton
                  icon={<IconFileImport size={16} />}
                  onClick={handleImportClick}
                  aria-label={t("games.importExport.importButton")}
                />
              </Tooltip>
              <ExpandableSearch
                value={searchQuery}
                onChange={setSearchQuery}
                placeholder={t("search")}
              />
              <Group gap="xs" wrap="nowrap" style={{ flexShrink: 0 }}>
                <FilterSegmentedControl
                  value={showFavorites}
                  onChange={setShowFavorites}
                  options={[
                    { value: "all", label: t("myGames.filters.all") },
                    {
                      value: "favorites",
                      label: t("myGames.filters.favorites"),
                    },
                  ]}
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
            <Group justify="space-between" wrap="wrap" gap="sm">
              <Group gap="sm">
                <ActionButton onClick={openCreateModal}>
                  {t("games.createButton")}
                </ActionButton>
                <ActionButton onClick={handleImportClick}>
                  {t("games.importExport.importButton")}
                </ActionButton>
              </Group>
              <Group gap="sm" wrap="wrap">
                <ExpandableSearch
                  value={searchQuery}
                  onChange={setSearchQuery}
                  placeholder={t("search")}
                />
                <FilterSegmentedControl
                  value={showFavorites}
                  onChange={setShowFavorites}
                  options={[
                    { value: "all", label: t("myGames.filters.all") },
                    {
                      value: "favorites",
                      label: t("myGames.filters.favorites"),
                    },
                  ]}
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
                      {t("myGames.empty.title")}
                    </Text>
                    <Text size="sm" c="gray.5" ta="center">
                      {t("myGames.empty.description")}
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
                        onClick={() => handleViewGame(game)}
                        onPlay={() => handlePlayGame(game)}
                        playLabel={t("myGames.play")}
                        hasSession={hasSession}
                        onContinue={
                          session
                            ? () => handleContinueGame(session)
                            : undefined
                        }
                        continueLabel={t("myGames.continue")}
                        onRestart={
                          session
                            ? () => handleRestartGame(game, session)
                            : undefined
                        }
                        restartLabel={t("myGames.restart")}
                        showVisibility
                        isFavorite={isFavorite(game)}
                        onToggleFavorite={() => handleToggleFavorite(game)}
                        favoriteLabel={t("myGames.favorite")}
                        unfavoriteLabel={t("myGames.unfavorite")}
                        actions={getCardActions(game)}
                        dateLabel={getDateLabel(game)}
                      />
                    );
                  })}
                </SimpleGrid>
              )
            ) : (games?.length ?? 0) === 0 ? (
              <Card shadow="sm" p="xl" radius="md" withBorder>
                <Stack align="center" gap="md" py="xl">
                  <IconMoodEmpty
                    size={48}
                    color="var(--mantine-color-gray-5)"
                  />
                  <Text c="gray.6" ta="center">
                    {t("myGames.empty.title")}
                  </Text>
                  <Text size="sm" c="gray.5" ta="center">
                    {t("myGames.empty.description")}
                  </Text>
                </Stack>
              </Card>
            ) : (
              <DataTable
                data={games ?? []}
                columns={columns}
                getRowKey={(game) => game.id || ""}
                onRowClick={handleViewGame}
                isLoading={false}
                fillHeight
                renderMobileCard={(game) => {
                  const { hasSession, session } = getGameSessionState(game);
                  return (
                    <GameCard
                      game={game}
                      onClick={() => handleViewGame(game)}
                      onPlay={() => handlePlayGame(game)}
                      playLabel={t("myGames.play")}
                      hasSession={hasSession}
                      onContinue={
                        session ? () => handleContinueGame(session) : undefined
                      }
                      continueLabel={t("myGames.continue")}
                      onRestart={
                        session
                          ? () => handleRestartGame(game, session)
                          : undefined
                      }
                      restartLabel={t("myGames.restart")}
                      showVisibility
                      isFavorite={isFavorite(game)}
                      onToggleFavorite={() => handleToggleFavorite(game)}
                      favoriteLabel={t("myGames.favorite")}
                      unfavoriteLabel={t("myGames.unfavorite")}
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
                    title={t("myGames.empty.title")}
                    description={t("myGames.empty.description")}
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
          onModalClose?.();
        }}
        onSponsor={() => {
          const game = rawGames?.find((g) => g.id === gameToView);
          if (game) {
            setGameToSponsor(game);
            openSponsorModal();
          }
        }}
      />

      <SponsorGameModal
        game={gameToSponsor}
        opened={sponsorModalOpened}
        onClose={() => {
          closeSponsorModal();
          setGameToSponsor(null);
        }}
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
      />
    </>
  );
}
