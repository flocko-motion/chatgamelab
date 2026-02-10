import { useTranslation } from "react-i18next";
import { useGameTheme } from "../theme";
import { translateErrorCode } from "@/common/lib/errorHelpers";
import { ErrorCodes } from "@/common/types/errorCodes";
import { ThemedText } from "./text-effects";
import classes from "./GamePlayer.module.css";

interface PlayerActionProps {
  text: string;
  error?: string;
  errorCode?: string;
  onRetry?: () => void;
}

// Map indicator type to display character
const INDICATOR_CHARS: Record<string, string> = {
  dot: "•",
  chevron: ">",
  pipe: "|",
  cursor: "▌",
  underscore: "_",
  none: "",
};

export function PlayerAction({
  text,
  error,
  errorCode,
  onRetry,
}: PlayerActionProps) {
  const { t } = useTranslation("common");
  const { theme, PlayerMessageWrapper } = useGameTheme();
  const indicator = theme.player.indicator ?? "chevron";
  const indicatorBlink = theme.player.indicatorBlink ?? false;

  const bubbleClasses = [
    classes.playerActionBubble,
    indicator === "none" && classes.noIndicator,
    error && classes.playerActionError,
  ]
    .filter(Boolean)
    .join(" ");

  const indicatorClasses = [
    classes.playerIndicator,
    indicatorBlink && classes.indicatorBlink,
  ]
    .filter(Boolean)
    .join(" ");

  const indicatorChar = INDICATOR_CHARS[indicator] || ">";

  const errorInfo = error && errorCode ? translateErrorCode(errorCode) : null;
  const errorMessage = errorInfo?.message || t("gamePlayer.error.sendFailed");
  const canRetry = onRetry && errorCode !== ErrorCodes.NO_API_KEY;

  return (
    <div className={classes.playerAction}>
      <div className={bubbleClasses}>
        {indicator !== "none" && (
          <span className={indicatorClasses}>{indicatorChar}</span>
        )}
        <span className={classes.playerActionText}>
          {PlayerMessageWrapper ? (
            <PlayerMessageWrapper text={text}>{text}</PlayerMessageWrapper>
          ) : (
            <ThemedText text={text} scope="playerMessages" />
          )}
        </span>
      </div>
      {error && (
        <div className={classes.playerActionErrorInfo}>
          <span className={classes.playerActionErrorText}>
            ⚠️ {errorMessage}
          </span>
          {canRetry && (
            <button
              className={classes.playerActionRetryButton}
              onClick={onRetry}
              type="button"
            >
              ↻ {t("gamePlayer.error.retry")}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
