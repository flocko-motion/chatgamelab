import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/providers/AuthProvider";
import { useBranding } from "@/providers/branding";
import { AdvancedLaunchPanel } from "./components/AdvancedLaunchPanel";
import { LaunchPanel } from "./components/LaunchPanel";
import { usePlaybackLaunch } from "./hooks/usePlaybackLaunch";
import { playbackURL, type PlaybackTarget } from "./playback";

/**
 * Every play route: the platform's launch panel until the session is had
 * (usePlaybackLaunch), then the playback-engine playing it in an iframe.
 */
export function PlaybackHost({ target }: { target: PlaybackTarget }) {
  const { i18n } = useTranslation();
  const branding = useBranding();
  const navigate = useNavigate();
  const { isLoading, isAuthenticated } = useAuth();
  const launch = usePlaybackLaunch(target, i18n.language, !isLoading);
  const { view } = launch;

  if (view.kind === "playing") {
    return (
      <iframe
        key={view.id}
        src={playbackURL(view.id, i18n.language)}
        title={branding.siteName}
        allow="microphone; autoplay; clipboard-write; fullscreen"
        style={{ flex: 1, width: "100%", border: 0, display: "block" }}
      />
    );
  }

  const back = isAuthenticated ? () => navigate({ to: "/games" }) : undefined;
  if (view.kind === "advanced") {
    return <AdvancedLaunchPanel game={view.game} onStart={launch.startWithPrompts} onBack={back} />;
  }

  return (
    <LaunchPanel
      view={view}
      onContinue={launch.continueSession}
      onStartAnew={launch.startAnew}
      onTryAgain={launch.tryAgain}
      onChoose={launch.choose}
      onBack={back}
    />
  );
}
