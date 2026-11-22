import { Center, Loader, Paper, Stack } from "@mantine/core";
import { useTranslation } from "react-i18next";
import { ActionButton, PlayGameButton, TextButton } from "@components/buttons";
import { BodyText, CardTitle, HelperText } from "@components/typography";
import { problemText, type Problem } from "@/common/lib/playability";
import type { LaunchView } from "../hooks/usePlaybackLaunch";

interface LaunchPanelProps {
  view: Exclude<LaunchView, { kind: "playing" }>;
  onContinue: (id: string) => void;
  onStartAnew: () => void;
  onTryAgain: () => void;
  /** Absent for a guest, who has no game list to go back to. */
  onBack?: () => void;
}

/** Problems trying again may get past: the server or the provider was away. */
const RETRYABLE: ReadonlyArray<Problem | "offline"> = [
  "offline",
  "unreachable",
  "failed",
];

/**
 * The platform's panel in the play route's place until the session plays:
 * starting or opening, why a game cannot start or go on, or, for a share
 * link's guest, continuing the game in progress or starting anew.
 */
export function LaunchPanel({
  view,
  onContinue,
  onStartAnew,
  onTryAgain,
  onBack,
}: LaunchPanelProps) {
  const { t } = useTranslation(["common", "playbackEngine"]);

  if (view.kind === "working") {
    return (
      <Center style={{ flex: 1 }} p="md">
        <Stack align="center" gap="sm" role="status">
          <Loader size="lg" />
          <HelperText>{t(`play.launch.${view.text}`)}</HelperText>
        </Stack>
      </Center>
    );
  }

  const back = onBack && (
    <TextButton onClick={onBack}>{t("play.launch.backToGames")}</TextButton>
  );

  if (view.kind === "resumable") {
    return (
      <Panel title={t("play.launch.inProgress")}>
        <BodyText>{t("play.launch.inProgressHint")}</BodyText>
        <Stack gap="xs" align="stretch">
          <PlayGameButton size="md" onClick={() => onContinue(view.id)}>
            {t("play.launch.continue")}
          </PlayGameButton>
          {view.startAnew && (
            <TextButton onClick={onStartAnew}>
              {t("play.launch.startAnew")}
            </TextButton>
          )}
        </Stack>
        {view.startAnew && view.usesPlay && (
          <HelperText>{t("play.launch.startAnewUsesPlay")}</HelperText>
        )}
        {!view.startAnew && (
          <HelperText>{t("play.launch.noPlaysLeft")}</HelperText>
        )}
        {back}
      </Panel>
    );
  }

  const reason =
    view.problem === "offline"
      ? t("play.launch.offline")
      : t(problemText(view.problem), { ns: "playbackEngine" });
  return (
    <Panel
      title={
        view.when === "start"
          ? t("play.launch.cannotStart")
          : t("play.launch.cannotOpen")
      }
    >
      <BodyText>{reason}</BodyText>
      {RETRYABLE.includes(view.problem) && (
        <ActionButton onClick={onTryAgain}>
          {t("play.launch.tryAgain")}
        </ActionButton>
      )}
      {back}
    </Panel>
  );
}

function Panel({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <Center style={{ flex: 1 }} p="md">
      <Paper withBorder radius="md" p="lg" shadow="sm" maw={440} w="100%">
        <Stack gap="md">
          <CardTitle>{title}</CardTitle>
          {children}
        </Stack>
      </Paper>
    </Center>
  );
}
