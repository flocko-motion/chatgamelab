import { useRef, type ReactNode } from "react";
import { Tooltip } from "@mantine/core";
import { IconArrowBigUpLineFilled, IconTools } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { GameView } from "@/api/types";
import { ListTable } from "@components/ListTable";
import { ActionButton, GenericIconButton } from "@components/buttons";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import { GAME_LIST_PRESETS } from "../lib/gameListPresets";
import { emptyListDescription, getGameDateLabel } from "../lib";
import { useGamesTable, type UseGamesTableOptions } from "../hooks/useGamesTable";
import { useGameActions, useGameColumns } from "./useGameColumns";
import { GameCard } from "./GameCard";
import { GameEditModal } from "./GameEditModal";
import { DeleteGameModal } from "./DeleteGameModal";
import { SponsorGameModal } from "./SponsorGameModal";
import { WorkshopBadges } from "./WorkshopBadges";

export type GamesTableProps = UseGamesTableOptions;

/**
 * A game list: one of the presets (all games, the workshop's
 * games) with its filters, orders, columns, actions and modals. The server
 * filters, sorts and pages; the query lives in the URL.
 */
export function GamesTable(props: GamesTableProps) {
  const { t, i18n } = useTranslation("common");
  const { t: tEmpty } = useTranslation(GAME_LIST_PRESETS[props.preset].empty.ns ?? "common");
  const { isMobile } = useResponsiveDesign();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const state = useGamesTable(props, { fileInputRef });
  const columns = useGameColumns(state);
  const actionsFor = useGameActions(state);
  const { config, list } = state;

  // An empty list says what narrowed it, when anything did; otherwise there
  // simply are no games.
  const narrowed =
    (config.list.toggles?.length ?? 0) > 0
      ? emptyListDescription(t, i18n.language, list.query.only, list.query.search)
      : null;
  const empty = narrowed
    ? { title: narrowed }
    : { title: tEmpty(config.empty.titleKey), description: tEmpty(config.empty.descriptionKey) };

  const renderCard = (game: GameView) => {
    const { hasSession, session } = state.getSessionState(game);
    const p = state.permissions(game);
    const favorites = config.columns.includes("favorite");
    return (
      <GameCard
        game={game}
        onEdit={() => state.view(game)}
        editLabel={t("editGame")}
        onOpen={(canStartNew) => state.openGame(game, session, canStartNew)}
        playLabel={t("allGames.play")}
        hasSession={hasSession}
        continueLabel={t("allGames.continue")}
        showVisibility={config.columns.includes("visibility")}
        isOwner={p.isOwner}
        isFavorite={favorites && state.isFavorite(game)}
        onToggleFavorite={favorites ? () => state.toggleFavorite(game) : undefined}
        favoriteLabel={t("allGames.favorite")}
        unfavoriteLabel={t("allGames.unfavorite")}
        actions={actionsFor(game)}
        dateLabel={getGameDateLabel(game, list.query.sortBy)}
        isWorkshopGame={state.isWorkshopGame(game)}
        extra={
          game.workshopId && config.columns.includes("workshop") ? (
            <WorkshopBadges workshopId={game.workshopId} />
          ) : undefined
        }
      />
    );
  };

  let createActions: ReactNode = null;
  if (config.canCreate) {
    createActions = isMobile ? (
      <>
        <Tooltip label={t("games.createButton")} withArrow>
          <GenericIconButton
            icon={<IconTools size={16} />}
            onClick={state.openCreate}
            variant="filled"
            color="accent"
            aria-label={t("games.createButton")}
          />
        </Tooltip>
        <Tooltip label={t("games.importExport.importButton")} withArrow>
          <GenericIconButton
            icon={<IconArrowBigUpLineFilled size={16} />}
            onClick={state.openImport}
            variant="filled"
            color="accent"
            aria-label={t("games.importExport.importButton")}
          />
        </Tooltip>
      </>
    ) : (
      <>
        <ActionButton onClick={state.openCreate} leftSection={<IconTools size={16} />}>
          {t("games.createButton")}
        </ActionButton>
        <ActionButton onClick={state.openImport} leftSection={<IconArrowBigUpLineFilled size={16} />}>
          {t("games.importExport.importButton")}
        </ActionButton>
      </>
    );
  }

  return (
    <>
      {config.canCreate && (
        <input
          type="file"
          ref={fileInputRef}
          onChange={state.importFile}
          accept=".cgl,.yaml,.yml"
          style={{ display: "none" }}
        />
      )}

      <ListTable
        list={list}
        columns={columns}
        getRowKey={(game) => game.id}
        renderCard={renderCard}
        layout="grid"
        onRowClick={state.view}
        actions={createActions}
        empty={empty}
        errorMessage={t("games.errors.loadFailed")}
      />

      <GameEditModal
        opened={state.createModal.opened}
        onClose={state.createModal.close}
        onCreate={state.createModal.submit}
        createLoading={state.createModal.loading}
        initialData={state.createModal.initialData}
        saveAsOwn={state.createModal.saveAsOwn}
      />

      <GameEditModal
        gameId={state.viewModal.gameId}
        opened={state.viewModal.opened}
        onClose={state.viewModal.close}
        isOwner={state.viewModal.isOwner}
        onSponsor={state.viewModal.onSponsor}
        showShareSection
        workshopId={state.viewModal.workshopId}
      />

      <DeleteGameModal
        opened={state.deleteModal.opened}
        onClose={state.deleteModal.close}
        onConfirm={state.deleteModal.confirm}
        gameName={state.deleteModal.game?.name ?? ""}
        loading={state.deleteModal.loading}
        isOwner={state.deleteModal.isOwner}
        isModeratorAction={state.deleteModal.isOthersWorkshopGame}
      />

      <SponsorGameModal
        game={state.sponsorModal.game}
        opened={state.sponsorModal.opened}
        onClose={state.sponsorModal.close}
      />
    </>
  );
}
