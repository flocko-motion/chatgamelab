import type { ReactNode } from "react";
import { Anchor, UnstyledButton } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { useGame } from "@/api/hooks";
import type { GameView, SessionView } from "@/api/types";
import { GameCover } from "@components/GameCover";
import { GenreChip } from "@components/GenreChip";
import { Critter } from "@components/Critter";
import { HelperText, ItemTitle, Kicker, SectionTitle } from "@components/typography";
import { formatRelativeTime } from "@/common/lib/formatters";
import classes from "./Dashboard.module.css";

/** A section's head: title, and a hint or a link to see everything. */
export function SectionHead({ title, hint, action }: { title: string; hint?: string; action?: ReactNode }) {
  return (
    <div className={classes.sectionHead}>
      <SectionTitle>{title}</SectionTitle>
      {action ?? (hint && <span className={classes.sectionHint}><Kicker>{hint}</Kicker></span>)}
    </div>
  );
}

/** "See all" at the end of a section's head. */
export function SeeAll({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <Anchor component="button" type="button" onClick={onClick}>
      {label} →
    </Anchor>
  );
}

/** A game in progress: its cover and genre (from the game), how far it got, Continue. */
function ContinueCard({ session, onOpen }: { session: SessionView; onOpen: () => void }) {
  const { t, i18n } = useTranslation("dashboard");
  const { data: game } = useGame(session.gameId);
  const name = session.gameName ?? t("untitledGame");
  return (
    <UnstyledButton className={`${classes.continueCard} ${classes.wobble}`} onClick={onOpen}>
      <div className={classes.continueCover}>
        <GameCover game={game ?? { id: session.gameId, name }} flush ratio="fill" />
      </div>
      <div className={classes.itemBody}>
        {game && <span><GenreChip genre={game.genre} size="sm" /></span>}
        <ItemTitle lines={1}>{name}</ItemTitle>
        <span className={classes.meta}>
          {t("continue.messages", { count: session.messagesSent })} ·{" "}
          {formatRelativeTime(session.modifiedAt, i18n.language)}
        </span>
        <span className={classes.continueGo}>{t("continue.go")} →</span>
      </div>
    </UnstyledButton>
  );
}

export function ContinueGames({ sessions, onOpen }: { sessions: SessionView[]; onOpen: (s: SessionView) => void }) {
  const { t } = useTranslation("dashboard");
  if (sessions.length === 0) return null;
  return (
    <section>
      <SectionHead title={t("continue.title")} hint={t("continue.hint")} />
      <div className={classes.continueGrid}>
        {sessions.map((session) => (
          <ContinueCard key={session.id} session={session} onOpen={() => onOpen(session)} />
        ))}
      </div>
    </section>
  );
}

/** A square tile for one of the user's own games. */
function OwnGameTile({ game, onOpen }: { game: GameView; onOpen: () => void }) {
  const { t, i18n } = useTranslation("dashboard");
  return (
    <UnstyledButton className={`${classes.itemCard} ${classes.wobble}`} onClick={onOpen}>
      <div className={classes.cover}>
        <GameCover game={game} flush ratio="1/1" />
        <span className={classes.genreOnCover}><GenreChip genre={game.genre} size="sm" /></span>
      </div>
      <div className={classes.itemBody}>
        <ItemTitle>{game.name}</ItemTitle>
        <span className={classes.monoMeta}>
          {t("popular.plays", { count: game.playCount })} ·{" "}
          {formatRelativeTime(game.meta?.modifiedAt ?? game.meta?.createdAt, i18n.language)}
        </span>
      </div>
    </UnstyledButton>
  );
}

export function OwnGames({
  games,
  loading,
  onOpen,
  onSeeAll,
}: {
  games: GameView[];
  loading: boolean;
  onOpen: (g: GameView) => void;
  onSeeAll: () => void;
}) {
  const { t } = useTranslation("dashboard");
  return (
    <section>
      <SectionHead
        title={t("ownGames.title")}
        action={games.length > 0 ? <SeeAll label={t("ownGames.viewAll")} onClick={onSeeAll} /> : undefined}
      />
      {!loading && games.length === 0 ? (
        <HelperText>{t("ownGames.empty")}</HelperText>
      ) : (
        <div className={classes.tiles}>
          {games.map((game) => (
            <OwnGameTile key={game.id} game={game} onOpen={() => onOpen(game)} />
          ))}
        </div>
      )}
    </section>
  );
}

/** Who made a game: their critter and name. */
function Creator({ game }: { game: GameView }) {
  const { t } = useTranslation("dashboard");
  const name = game.creatorName ?? t("someone");
  return (
    <span className={classes.meta}>
      <Critter seed={game.creatorId ?? name} size={20} />
      {t("fresh.by", { name })}
    </span>
  );
}

export function FreshGames({
  games,
  onPlay,
  onSeeAll,
}: {
  games: GameView[];
  onPlay: (g: GameView) => void;
  onSeeAll: () => void;
}) {
  const { t } = useTranslation("dashboard");
  if (games.length === 0) return null;
  return (
    <section>
      <SectionHead title={t("fresh.title")} action={<SeeAll label={t("fresh.viewAll")} onClick={onSeeAll} />} />
      <div className={classes.discoveries}>
        {games.map((game) => (
          <UnstyledButton key={game.id} className={`${classes.itemCard} ${classes.wobble}`} onClick={() => onPlay(game)}>
            <div className={classes.cover}>
              <GameCover game={game} flush />
              <span className={classes.genreOnCover}><GenreChip genre={game.genre} size="sm" /></span>
            </div>
            <div className={classes.itemBody}>
              <ItemTitle>{game.name}</ItemTitle>
              <Creator game={game} />
            </div>
          </UnstyledButton>
        ))}
      </div>
    </section>
  );
}

export function PopularGames({ games, onPlay }: { games: GameView[]; onPlay: (g: GameView) => void }) {
  const { t } = useTranslation("dashboard");
  if (games.length === 0) return null;
  return (
    <section>
      <SectionHead title={t("popular.title")} hint={t("popular.hint")} />
      <ol className={classes.ranked} style={{ listStyle: "none", padding: 0, margin: 0 }}>
        {games.map((game, index) => (
          <li key={game.id}>
            <UnstyledButton className={classes.rank} onClick={() => onPlay(game)}>
              <span className={classes.rankNo} aria-hidden>{index + 1}</span>
              <span className={classes.rankCover}><GameCover game={game} flush ratio="1/1" /></span>
              <span className={classes.rankName}>
                <ItemTitle>{game.name}</ItemTitle>
                <GenreChip genre={game.genre} size="sm" />
              </span>
              <span className={classes.monoMeta}>{t("popular.plays", { count: game.playCount })}</span>
            </UnstyledButton>
          </li>
        ))}
      </ol>
    </section>
  );
}
