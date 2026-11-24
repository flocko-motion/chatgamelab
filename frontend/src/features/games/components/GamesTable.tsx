import { useRef } from "react";
import { useTranslation } from "react-i18next";
import type { GameView } from "@/api/types";
import { ListTable } from "@components/ListTable";
import { GAME_LIST_PRESETS } from "../lib/gameListPresets";
import { emptyListDescription, getGameDateLabel } from "../lib";
import { useGamesTable, type UseGamesTableOptions } from "../hooks/useGamesTable";
import { useGameColumns } from "./useGameColumns";
import { GameCard } from "./GameCard";
import { GameEditor } from "./editor/GameEditor";
import { SponsorGameModal } from "./SponsorGameModal";
import { WorkshopBadges } from "./WorkshopBadges";
import { NewGameMenu } from "./NewGameMenu";
import { useGameFilterLook } from "../hooks/useGameFilterLook";

export type GamesTableProps = UseGamesTableOptions;

/**
 * A game list: one of the presets (all games, the workshop's
 * games) with its filters, orders, columns, actions and modals. The server
 * filters, sorts and pages; the query lives in the URL.
 */
export function GamesTable(props: GamesTableProps) {
  const { t, i18n } = useTranslation("common");
  const { t: tEmpty } = useTranslation(GAME_LIST_PRESETS[props.preset].empty.ns ?? "common");
  const fileInputRef = useRef<HTMLInputElement>(null);
  const state = useGamesTable(props, { fileInputRef });
  const columns = useGameColumns(state);
  const { config, list } = state;
  const filterLook = useGameFilterLook();

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
        editLabel={p.canEdit ? t("editGame") : t("games.editor.open")}
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

  const createActions = config.canCreate ? (
    <NewGameMenu onCreate={state.openCreate} onImport={state.openImport} />
  ) : null;

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
        optionLook={filterLook}
        empty={empty}
        errorMessage={t("games.errors.loadFailed")}
      />

      <GameEditor
        opened={state.createModal.opened}
        onClose={state.createModal.close}
        onCreate={state.createModal.submit}
        createLoading={state.createModal.loading}
        initialData={state.createModal.initialData}
        saveAsOwn={state.createModal.saveAsOwn}
      />

      <GameEditor
        gameId={state.viewModal.gameId}
        opened={state.viewModal.opened}
        onClose={state.viewModal.close}
        isOwner={state.viewModal.isOwner}
        isOthersWorkshopGame={state.viewModal.isOthersWorkshopGame}
        onSponsor={state.viewModal.onSponsor}
        onCopy={state.viewModal.copy}
        showShareSection
        workshopId={state.viewModal.workshopId}
      />

      <SponsorGameModal
        game={state.sponsorModal.game}
        opened={state.sponsorModal.opened}
        onClose={state.sponsorModal.close}
      />
    </>
  );
}
