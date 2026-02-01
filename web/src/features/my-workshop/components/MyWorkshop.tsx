import React, { useState } from "react";
import {
  Stack,
  Group,
  Alert,
  SimpleGrid,
  Text,
  Badge,
  Tooltip,
  Box,
} from "@mantine/core";
import { useDisclosure, useMediaQuery } from "@mantine/hooks";
import { useTranslation } from "react-i18next";
import { useNavigate } from "@tanstack/react-router";
import {
  IconAlertCircle,
  IconMoodEmpty,
  IconCopy,
  IconDownload,
  IconSchool,
  IconWorld,
  IconUser,
  IconEye,
} from "@tabler/icons-react";
import {
  PlayGameButton,
  EditIconButton,
  DeleteIconButton,
  GenericIconButton,
  TextButton,
} from "@components/buttons";
import {
  DataTable,
  DataTableEmptyState,
  type DataTableColumn,
} from "@components/DataTable";
import { DimmedLoader } from "@components/LoadingAnimation";
import type { ObjGame, DbUserSessionWithGame } from "@/api/generated";
import type { CreateGameFormData } from "@/features/games/types";
import {
  GameEditModal,
  DeleteGameModal,
  GameCard,
  type GameCardAction,
} from "@/features/games";
import { useModals } from "@mantine/modals";
import { useAuth } from "@/providers/AuthProvider";
import { hasRole, Role } from "@/common/lib/roles";
import {
  isWorkshopGame,
  type GameFilter,
  type WorkshopSettings,
} from "../types";
import { useWorkshopGames } from "../hooks";
import { useWorkshopEvents, useGamesCacheUpdater } from "@/api/hooks";
import { WorkshopHeader } from "./WorkshopHeader";
import { WorkshopControls } from "./WorkshopControls";
import { WorkshopLoadingSkeleton } from "./WorkshopLoadingSkeleton";
import { WorkshopEmptyState } from "./WorkshopEmptyState";

export function MyWorkshop() {
  const { t } = useTranslation("common");
  const { t: tWorkshop } = useTranslation("myWorkshop");
  const isMobile = useMediaQuery("(max-width: 48em)") ?? false;
  const navigate = useNavigate();
  const modals = useModals();
  const { backendUser, retryBackendFetch } = useAuth();
  const { addGameToCache, updateGameInCache, removeGameFromCache } =
    useGamesCacheUpdater();

  // User info
  const canEditAllWorkshopGames =
    hasRole(backendUser, Role.Head) || hasRole(backendUser, Role.Staff);
  const currentUserId = backendUser?.id;
  const workshopName = backendUser?.role?.workshop?.name;
  const organizationName = backendUser?.role?.institution?.name;

  // Workshop settings
  const workshop = backendUser?.role?.workshop;
  const workshopSettings: WorkshopSettings = {
    showPublicGames: workshop?.showPublicGames ?? false,
    showOtherParticipantsGames: workshop?.showOtherParticipantsGames ?? true,
  };

  // UI State
  const [
    createModalOpened,
    { open: openCreateModal, close: closeCreateModal },
  ] = useDisclosure(false);
  const [
    deleteModalOpened,
    { open: openDeleteModal, close: closeDeleteModal },
  ] = useDisclosure(false);
  const [viewModalOpened, { open: openViewModal, close: closeViewModal }] =
    useDisclosure(false);
  const [gameToDelete, setGameToDelete] = useState<ObjGame | null>(null);
  const [gameToView, setGameToView] = useState<string | null>(null);
  const [gameToViewReadOnly, setGameToViewReadOnly] = useState(false);
  const [sortValue, setSortValue] = useState("modifiedAt-desc");
  const [gameFilter, setGameFilter] = useState<GameFilter>("all");
  const [searchQuery, setSearchQuery] = useState("");

  // Games hook
  const {
    games,
    rawGames,
    sortField,
    isLoading,
    isFetching,
    sessionsLoading,
    error,
    isCreating,
    isDeleting,
    isCloning,
    getPermissions,
    getSessionState,
    fileInputRef,
    handleCreateGame,
    handleDeleteGame,
    handleExportGame,
    handleCloneGame,
    handleDeleteSession,
    triggerImportClick,
    handleImportFile,
  } = useWorkshopGames({
    currentUserId,
    currentWorkshopId: workshop?.id,
    canEditAllWorkshopGames,
    workshopSettings,
    gameFilter,
    sortValue,
    searchQuery,
  });

  // Subscribe to real-time workshop events (settings changes and game updates)
  useWorkshopEvents({
    workshopId: workshop?.id,
    onSettingsUpdate: retryBackendFetch,
    // Update cache with single game instead of refetching entire list
    onGameCreated: addGameToCache,
    onGameUpdated: updateGameInCache,
    onGameDeleted: removeGameFromCache,
  });

  // Handlers
  const onCreateGame = async (data: CreateGameFormData) => {
    try {
      await handleCreateGame(data);
      closeCreateModal();
    } catch {
      // Error handled by mutation
    }
  };

  const handleEditGame = (game: ObjGame) => {
    const { canEdit } = getPermissions(game);
    if (game.id && canEdit) {
      setGameToView(game.id);
      setGameToViewReadOnly(false);
      openViewModal();
    }
  };

  const handleViewGame = (game: ObjGame) => {
    const { canEdit } = getPermissions(game);
    if (game.id) {
      setGameToView(game.id);
      setGameToViewReadOnly(!canEdit);
      openViewModal();
    }
  };

  const handleCopyFromModal = async () => {
    if (!gameToView) return;
    try {
      const newGame = await handleCloneGame(gameToView);
      closeViewModal();
      setGameToView(null);
      if (newGame.id) {
        setGameToView(newGame.id);
        setGameToViewReadOnly(false);
        openViewModal();
      }
    } catch {
      // Error handled by mutation
    }
  };

  const handleDeleteClick = (game: ObjGame) => {
    const { canDelete } = getPermissions(game);
    if (canDelete) {
      setGameToDelete(game);
      openDeleteModal();
    }
  };

  const handleConfirmDelete = async () => {
    if (!gameToDelete?.id) return;
    try {
      await handleDeleteGame(gameToDelete.id);
      closeDeleteModal();
      setGameToDelete(null);
    } catch {
      // Error handled by mutation
    }
  };

  const handleCopyGame = (game: ObjGame) => {
    if (!game.id) return;
    modals.openConfirmModal({
      title: t("myGames.copyConfirm.title"),
      children: (
        <Text size="sm">
          {t("myGames.copyConfirm.message", {
            name: game.name || t("sessions.untitledGame"),
          })}
        </Text>
      ),
      labels: {
        confirm: t("myGames.copyConfirm.confirm"),
        cancel: t("cancel"),
      },
      onConfirm: async () => {
        const newGame = await handleCloneGame(game.id!);
        if (newGame.id) {
          setGameToView(newGame.id);
          openViewModal();
        }
      },
    });
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
          await handleDeleteSession(session.id!);
        } catch {
          // Session may have been deleted already, ignore and continue
        }
        navigate({ to: "/games/$gameId/play", params: { gameId: game.id! } });
      },
    });
  };

  const handleFileSelect = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0];
    if (!file) return;
    try {
      const newGameId = await handleImportFile(file);
      if (newGameId) {
        setGameToView(newGameId);
        openViewModal();
      }
    } catch {
      // Error handled
    }
    event.target.value = "";
  };

  // Render helpers
  const renderPlayButton = (game: ObjGame) => {
    const { hasSession, session } = getSessionState(game);
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

  const getDateLabel = (game: ObjGame) => {
    const dateValue =
      sortField === "createdAt" ? game.meta?.createdAt : game.meta?.modifiedAt;
    return dateValue ? new Date(dateValue).toLocaleDateString() : undefined;
  };

  const getCardActions = (game: ObjGame): GameCardAction[] => {
    const { canEdit, canDelete } = getPermissions(game);
    const actions: GameCardAction[] = [];

    if (canEdit) {
      actions.push({
        key: "edit",
        icon: null,
        label: t("editGame"),
        onClick: () => handleEditGame(game),
      });
    }
    actions.push({
      key: "copy",
      icon: <IconCopy size={16} />,
      label: t("copyGame"),
      onClick: () => handleCopyGame(game),
    });
    actions.push({
      key: "export",
      icon: <IconDownload size={16} />,
      label: t("games.importExport.exportButton"),
      onClick: () => handleExportGame(game),
    });
    if (canDelete) {
      actions.push({
        key: "delete",
        icon: null,
        label: t("deleteGame"),
        onClick: () => handleDeleteClick(game),
      });
    }
    return actions;
  };

  const getGameBadge = (game: ObjGame) => {
    const { isOwner } = getPermissions(game);
    if (isOwner) {
      return (
        <Badge
          size="xs"
          color="violet"
          variant="light"
          leftSection={<IconUser size={10} />}
        >
          {tWorkshop("filters.mine")}
        </Badge>
      );
    }
    if (game.public) {
      return (
        <Badge
          size="xs"
          color="green"
          variant="light"
          leftSection={<IconWorld size={10} />}
        >
          {t("games.visibility.public")}
        </Badge>
      );
    }
    if (game.workshopId) {
      return (
        <Badge
          size="xs"
          color="accent"
          variant="light"
          leftSection={<IconSchool size={10} />}
        >
          {tWorkshop("filters.workshop")}
        </Badge>
      );
    }
    return null;
  };

  // Table columns
  const columns: DataTableColumn<ObjGame>[] = [
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
      key: "creator",
      header: t("games.fields.creator"),
      width: 150,
      render: (game) => {
        const { isOwner } = getPermissions(game);
        return (
          <Text size="sm" c="gray.6" lineClamp={1}>
            {isOwner ? tWorkshop("you") : game.creatorName || "-"}
          </Text>
        );
      },
    },
    {
      key: "type",
      header: tWorkshop("gameType"),
      width: 130,
      render: (game) => getGameBadge(game),
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
      width: 220,
      render: (game) => {
        const { canEdit, canDelete } = getPermissions(game);
        return (
          <Group gap="md" onClick={(e) => e.stopPropagation()} wrap="nowrap" justify="flex-end">
            <Box style={{ width: 100, flexShrink: 0 }}>
              {renderPlayButton(game)}
            </Box>
            <SimpleGrid cols={2} spacing={4}>
              {canEdit ? (
                <Tooltip label={t("editGame")} withArrow>
                  <EditIconButton
                    onClick={() => handleEditGame(game)}
                    aria-label={t("edit")}
                  />
                </Tooltip>
              ) : (
                <Tooltip label={t("viewGame")} withArrow>
                  <GenericIconButton
                    icon={<IconEye size={16} />}
                    onClick={() => handleViewGame(game)}
                    aria-label={t("viewGame")}
                  />
                </Tooltip>
              )}
              <Tooltip label={t("copyGame")} withArrow>
                <GenericIconButton
                  icon={<IconCopy size={16} />}
                  onClick={() => handleCopyGame(game)}
                  aria-label={t("copyGame")}
                />
              </Tooltip>
              <Tooltip label={t("games.importExport.exportButton")} withArrow>
                <GenericIconButton
                  icon={<IconDownload size={16} />}
                  onClick={() => handleExportGame(game)}
                  aria-label={t("games.importExport.exportButton")}
                />
              </Tooltip>
              {canDelete ? (
                <Tooltip label={t("deleteGame")} withArrow>
                  <DeleteIconButton
                    onClick={() => handleDeleteClick(game)}
                    aria-label={t("delete")}
                  />
                </Tooltip>
              ) : (
                <Box />
              )}
            </SimpleGrid>
          </Group>
        );
      },
    },
  ];

  // Loading states
  const hasData = rawGames !== undefined;
  const isInitialLoading = !hasData && (isLoading || sessionsLoading);
  const isRefetching = isFetching && hasData;

  if (isInitialLoading) {
    return <WorkshopLoadingSkeleton isMobile={isMobile} />;
  }

  if (error) {
    return (
      <Alert
        icon={<IconAlertCircle size={16} />}
        title={t("error")}
        color="red"
      >
        {t("games.errors.loadFailed")}
      </Alert>
    );
  }

  const renderGameCard = (game: ObjGame) => {
    const { hasSession, session } = getSessionState(game);
    const { isOwner } = getPermissions(game);
    return (
      <GameCard
        key={game.id}
        game={game}
        onClick={() => handleViewGame(game)}
        onPlay={() => handlePlayGame(game)}
        playLabel={t("myGames.play")}
        hasSession={hasSession}
        onContinue={session ? () => handleContinueGame(session) : undefined}
        continueLabel={t("myGames.continue")}
        onRestart={session ? () => handleRestartGame(game, session) : undefined}
        restartLabel={t("myGames.restart")}
        showCreator
        isOwner={isOwner}
        creatorLabel={tWorkshop("you")}
        actions={getCardActions(game)}
        dateLabel={getDateLabel(game)}
        isWorkshopGame={isWorkshopGame(game, workshop?.id)}
      />
    );
  };

  return (
    <>
      <Stack
        gap="lg"
        h={{ base: "calc(100vh - 180px)", sm: "calc(100vh - 280px)" }}
        style={{ overflow: "hidden" }}
      >
        <Stack gap="md" style={{ flexShrink: 0 }}>
          <WorkshopHeader
            workshopName={workshopName}
            organizationName={organizationName}
          />
          <input
            type="file"
            ref={fileInputRef}
            onChange={handleFileSelect}
            accept=".yaml,.yml"
            style={{ display: "none" }}
          />
          <WorkshopControls
            searchQuery={searchQuery}
            onSearchChange={setSearchQuery}
            gameFilter={gameFilter}
            onFilterChange={setGameFilter}
            sortValue={sortValue}
            onSortChange={setSortValue}
            onCreateClick={openCreateModal}
            onImportClick={triggerImportClick}
            hasGames={(rawGames?.length ?? 0) > 0}
          />
        </Stack>

        <Box style={{ flex: 1, minHeight: 0, overflow: "auto" }}>
          <DimmedLoader visible={isRefetching} loaderSize="lg">
            {isMobile ? (
              (games?.length ?? 0) === 0 ? (
                <WorkshopEmptyState />
              ) : (
                <SimpleGrid cols={1} spacing="md">
                  {games?.map(renderGameCard)}
                </SimpleGrid>
              )
            ) : (games?.length ?? 0) === 0 ? (
              <WorkshopEmptyState />
            ) : (
              <DataTable
                data={games ?? []}
                columns={columns}
                getRowKey={(game) => game.id || ""}
                onRowClick={handleViewGame}
                isLoading={false}
                fillHeight
                renderMobileCard={renderGameCard}
                emptyState={
                  <DataTableEmptyState
                    icon={
                      <IconMoodEmpty
                        size={48}
                        color="var(--mantine-color-gray-5)"
                      />
                    }
                    title={tWorkshop("empty.title")}
                    description={tWorkshop("empty.description")}
                  />
                }
              />
            )}
          </DimmedLoader>
        </Box>
      </Stack>

      <GameEditModal
        opened={createModalOpened}
        onClose={closeCreateModal}
        onCreate={onCreateGame}
        createLoading={isCreating}
      />
      <GameEditModal
        gameId={gameToView}
        opened={viewModalOpened}
        onClose={() => {
          closeViewModal();
          setGameToView(null);
        }}
        readOnly={gameToViewReadOnly}
        onCopy={gameToViewReadOnly ? handleCopyFromModal : undefined}
        copyLoading={isCloning}
      />
      <DeleteGameModal
        opened={deleteModalOpened}
        onClose={() => {
          closeDeleteModal();
          setGameToDelete(null);
        }}
        onConfirm={handleConfirmDelete}
        gameName={gameToDelete?.name ?? ""}
        loading={isDeleting}
      />
    </>
  );
}
