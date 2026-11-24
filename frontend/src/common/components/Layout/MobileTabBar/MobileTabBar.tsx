import { UnstyledButton } from "@mantine/core";
import { IconCirclePlus, IconHome, IconLayoutGrid, IconUserCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useMobileTabs, type MobileTabKey } from "@/common/hooks/useMobileTabs";
import classes from "./MobileTabBar.module.css";

const ICONS: Record<MobileTabKey, typeof IconHome> = {
  home: IconHome,
  games: IconLayoutGrid,
  create: IconCirclePlus,
  me: IconUserCircle,
};

const LABELS: Record<MobileTabKey, string> = {
  home: "dashboard",
  games: "allGames",
  create: "createGame",
  me: "me",
};

/**
 * MobileTabBar - the main menu on a phone, at the bottom of the screen
 *
 * Dashboard, Games, Create Game and Me (the profile), one tap away. All tabs
 * look alike; the current one is an ink pill. The header's burger keeps the
 * full menu.
 */
export function MobileTabBar() {
  const { t } = useTranslation("navigation");
  const tabs = useMobileTabs();
  return (
    <nav className={classes.bar} aria-label={t("mainMenu")}>
      {tabs.map((tab) => {
        const Icon = ICONS[tab.key];
        return (
          <UnstyledButton
            key={tab.key}
            className={classes.tab}
            data-active={tab.active || undefined}
            aria-current={tab.active ? "page" : undefined}
            onClick={tab.open}
          >
            <span className={classes.icon}>
              <Icon size={22} stroke={1.8} aria-hidden />
            </span>
            <span className={classes.label}>{t(LABELS[tab.key])}</span>
          </UnstyledButton>
        );
      })}
    </nav>
  );
}
