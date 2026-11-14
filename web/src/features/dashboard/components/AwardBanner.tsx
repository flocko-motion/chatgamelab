import { useState } from "react";
import { Box, Button, CloseButton, Paper, Text } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { useCurrentAnnouncement } from "@/api/hooks";
import classes from "./AwardBanner.module.css";

const DISMISS_KEY_PREFIX = "cgl.announcement.dismissed.";

// Storage can be missing or throw (private mode, blocked site data).
// Any failure means "not dismissed", so the notice simply shows again.
function readDismissed(id: string): boolean {
  try {
    return window.localStorage.getItem(DISMISS_KEY_PREFIX + id) === "1";
  } catch {
    return false;
  }
}

function writeDismissed(id: string) {
  try {
    window.localStorage.setItem(DISMISS_KEY_PREFIX + id, "1");
  } catch {
    // Nothing to do: the notice is hidden for this page view anyway.
  }
}

/**
 * Time-limited dashboard notice for the Dieter Baacke Prize 2026.
 * Whether and when it shows is decided by the server (GET /announcements/current),
 * so it appears at the embargo time and disappears after the end date without
 * a deployment. The texts live in the "dashboard" locale namespace.
 */
export function AwardBanner() {
  const { t } = useTranslation("dashboard");
  const { data } = useCurrentAnnouncement();
  const [dismissedNow, setDismissedNow] = useState(false);

  if (!data?.active || !data.id || !data.link) return null;
  if (dismissedNow || readDismissed(data.id)) return null;

  const id = data.id;
  const handleClose = () => {
    writeDismissed(id);
    setDismissedNow(true);
  };

  return (
    <Paper
      component="section"
      role="region"
      aria-label={t("announcement.award.regionLabel")}
      withBorder
      shadow="sm"
      className={classes.banner}
    >
      <span className={classes.trophy} aria-hidden="true">
        🏆
      </span>
      <Box className={classes.body}>
        <Text fw={700} c="gray.9" className={classes.title}>
          {t("announcement.award.title")}
        </Text>
        <Text c="gray.8" size="sm">
          {t("announcement.award.text")}
        </Text>
      </Box>
      <Button
        component="a"
        href={data.link}
        target="_blank"
        rel="noopener noreferrer"
        color="accent.9"
        className={classes.cta}
      >
        {t("announcement.award.cta")}
        <span className={classes.srOnly}>
          {" "}
          {t("announcement.award.newTab")}
        </span>
      </Button>
      <CloseButton
        aria-label={t("announcement.award.close")}
        onClick={handleClose}
        className={classes.close}
        c="gray.8"
      />
    </Paper>
  );
}
