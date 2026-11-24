import { Text, Tooltip } from "@mantine/core";
import { IconStar, IconStarFilled } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { GameView } from "@/api/types";
import type { GameListSort } from "@/api/hooks";
import type { ListColumn } from "@components/ListTable";
import { GenericIconButton } from "@components/buttons";
import type { GameColumnKey } from "../lib/gameListPresets";
import { gameListDate } from "../lib";
import type { GamesTableState } from "../hooks/useGamesTable";
import { GamePlayButtons } from "./GamePlayButtons";
import { WorkshopBadges } from "./WorkshopBadges";
import { GameBadges, GameNameCell, VisibilityBadge } from "./GameListCells";

type GameColumn = ListColumn<GameView, GameListSort>;

/** The columns of a game list, in the order its preset names them. */
export function useGameColumns(state: GamesTableState): GameColumn[] {
  const { t } = useTranslation("common");
  const { sortBy } = state.list.query;

  const columns: Record<GameColumnKey, GameColumn | null> = {
    favorite: {
      key: "favorite",
      header: "",
      width: 40,
      interactive: true,
      // a shortcut to the "favorites" toggle of the filters
      headerToggle: {
        active: state.list.query.only.includes("favorites"),
        onToggle: () => state.list.toggleOnly("favorites"),
        icon: <IconStar size={18} />,
        activeIcon: <IconStarFilled size={18} color="var(--mantine-color-yellow-5)" />,
        label: t("allGames.onlyFavorites"),
      },
      render: (game) => {
        const label = state.isFavorite(game) ? t("allGames.unfavorite") : t("allGames.favorite");
        return (
          <Tooltip label={label} withArrow>
            <GenericIconButton
              icon={
                state.isFavorite(game) ? (
                  <IconStarFilled size={18} color="var(--mantine-color-yellow-5)" />
                ) : (
                  <IconStar size={18} />
                )
              }
              onClick={() => state.toggleFavorite(game)}
              aria-label={label}
            />
          </Tooltip>
        );
      },
    },
    name: {
      key: "name",
      header: t("games.fields.name"),
      sortBy: "name",
      render: (game) => <GameNameCell game={game} isOwner={state.permissions(game).isOwner} />,
    },
    creator: {
      key: "creator",
      header: t("games.fields.creator"),
      width: 180,
      sortBy: "creator",
      visibleFrom: "md",
      render: (game) => (
        <Text size="sm" c="gray.6" lineClamp={1}>
          {game.creatorName || "-"}
        </Text>
      ),
    },
    // workshopId comes only with games whose settings the caller may see.
    workshop: {
      key: "workshop",
      header: t("allGames.workshop"),
      width: 170,
      visibleFrom: "lg",
      render: (game) => (game.workshopId ? <WorkshopBadges workshopId={game.workshopId} /> : null),
    },
    badges: {
      key: "badges",
      header: t("games.fields.visibility"),
      width: 160,
      render: (game) => <GameBadges game={game} isWorkshopGame={state.isWorkshopGame(game)} />,
    },
    playCount: {
      key: "playCount",
      header: t("games.fields.playCount"),
      width: 90,
      sortBy: "playCount",
      visibleFrom: "md",
      render: (game) => (
        <Text size="sm" c="gray.6" ta="center">
          {game.playCount}
        </Text>
      ),
    },
    visibility: {
      key: "visibility",
      header: t("games.fields.visibility"),
      width: 150,
      sortBy: "visibility",
      render: (game) => <VisibilityBadge isPublic={game.public} />,
    },
    date: {
      key: "date",
      header: sortBy === "createdAt" ? t("games.fields.created") : t("games.fields.modified"),
      width: 110,
      // one header for both dates: Modified ↓ ↑, then Created ↓ ↑
      sortBy: ["modifiedAt", "createdAt"],
      render: (game) => {
        const value = gameListDate(game, sortBy);
        const date = value ? new Date(value) : null;
        return (
          <Tooltip label={date?.toLocaleString()} withArrow disabled={!date}>
            <Text size="sm" c="gray.6">
              {date ? date.toLocaleDateString() : "-"}
            </Text>
          </Tooltip>
        );
      },
    },
    actions: {
      key: "actions",
      header: t("actions"),
      width: 280,
      interactive: true,
      render: (game) => {
        const { hasSession, session } = state.getSessionState(game);
        return (
          <div style={{ width: 140 }}>
              <GamePlayButtons
                game={game}
                hasSession={hasSession}
                session={session}
                onPlay={(game) => state.openGame(game, undefined, true)}
                onContinue={state.continueGame}
                onRestart={state.restartGame}
                labels={{
                  play: t("allGames.play"),
                  continue: t("allGames.continue"),
                  restart: t("allGames.restart"),
                }}
              />
          </div>
        );
      },
    },
  };

  return state.config.columns
    .map((key) => columns[key])
    .filter((c): c is GameColumn => c !== null);
}
