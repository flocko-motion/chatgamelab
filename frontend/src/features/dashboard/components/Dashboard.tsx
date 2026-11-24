import { Anchor } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { EXTERNAL_LINKS } from "@/config/externalLinks";
import { HELP_LINKS } from "@/config/helpLinks";
import { useAdmin } from "@/common/hooks/useAdmin";
import { useDashboard } from "../hooks/useDashboard";
import { Hero } from "./Hero";
import { ContinueGames, FreshGames, OwnGames, PopularGames } from "./GameSections";
import { LabPanel } from "./LabPanel";
import classes from "./Dashboard.module.css";

/**
 * The dashboard: what shall we build today (prompt), games in progress, the
 * user's games and numbers, new and popular games of others.
 */
export function DashboardContent() {
  const { t } = useTranslation("dashboard");
  const { isAdmin } = useAdmin();
  const data = useDashboard();
  const { go } = data;

  return (
    <div className={classes.page}>
      <Hero name={data.firstName} onCreate={go.createAbout} />

      <ContinueGames sessions={data.sessions} onOpen={go.session} />

      <div className={classes.split}>
        {isAdmin ? (
          <FreshGames games={data.freshGames} onPlay={go.play} onSeeAll={go.gallery} />
        ) : (
          <OwnGames games={data.ownGames} loading={data.ownGamesLoading} onOpen={go.open} onSeeAll={go.ownGames} />
        )}
        <LabPanel data={data} />
      </div>

      {!isAdmin && <FreshGames games={data.freshGames} onPlay={go.play} onSeeAll={go.gallery} />}

      <PopularGames games={data.popularGames} onPlay={go.play} />

      <footer className={classes.links}>
        <Anchor href={HELP_LINKS.EDUCATOR_INFO} target="_blank" rel="noopener noreferrer">
          {t("links.educators")}
        </Anchor>
        <Anchor href={EXTERNAL_LINKS.CHATGAMELAB.href} target="_blank" rel="noopener noreferrer">
          {t("links.about")}
        </Anchor>
        <Anchor href={EXTERNAL_LINKS.JFF.href} target="_blank" rel="noopener noreferrer">
          {t("links.jff")}
        </Anchor>
      </footer>
    </div>
  );
}
