import { Anchor } from "@mantine/core";
import { useBranding } from "@/providers/branding";
import { useTranslation } from "react-i18next";
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
  const branding = useBranding();
  const { isAdmin } = useAdmin();
  const data = useDashboard();
  const { go } = data;

  return (
    <div className={classes.page}>
      <Hero name={data.firstName} onCreate={go.createAbout} />

      <ContinueGames sessions={data.sessions} onOpen={go.session} />

      <div className={classes.split}>
        {isAdmin ? (
          <FreshGames games={data.freshGames} card={data.card} onSeeAll={go.gallery} />
        ) : (
          <OwnGames games={data.ownGames} loading={data.ownGamesLoading} card={data.card} onSeeAll={go.ownGames} />
        )}
        <LabPanel data={data} />
      </div>

      {!isAdmin && <FreshGames games={data.freshGames} card={data.card} onSeeAll={go.gallery} />}

      <PopularGames games={data.popularGames} onPlay={go.play} />

      <footer className={classes.links}>
        {branding.educatorInfoUrl && (
          <Anchor href={branding.educatorInfoUrl} target="_blank" rel="noopener noreferrer">
            {t("links.educators")}
          </Anchor>
        )}
        {branding.websiteUrl && (
          <Anchor href={branding.websiteUrl} target="_blank" rel="noopener noreferrer">
            {t("links.about")}
          </Anchor>
        )}
        {branding.operatorUrl && branding.operatorName && (
          <Anchor href={branding.operatorUrl} target="_blank" rel="noopener noreferrer">
            {branding.operatorName}
          </Anchor>
        )}
      </footer>
    </div>
  );
}
