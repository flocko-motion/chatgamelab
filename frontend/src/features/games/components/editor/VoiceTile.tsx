import { Loader } from "@mantine/core";
import { IconPlayerPlayFilled, IconPlayerStopFilled } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import type { VoicePreview } from "../../hooks/useVoicePreview";
import { voiceBars } from "../../lib/voiceBars";
import classes from "./VoiceTile.module.css";

/**
 * The game's voice in the cover's shape: a waveform with a play button in the
 * middle. A click plays the sample, wherever the tile sits (inside an
 * overview card it plays without opening the card); while it plays, the
 * bars move and the button stops it.
 */
export function VoiceTile({ preview, style }: { preview: VoicePreview; style: string }) {
  const { t } = useTranslation("common");
  const busy = preview.loading || preview.renewing;
  return (
    <button
      type="button"
      className={`${classes.tile} ${preview.playing ? classes.playing : ""}`}
      aria-label={preview.playing ? t("games.editor.sound.stop") : t("games.editor.sound.play")}
      disabled={busy}
      onClick={(e) => {
        e.stopPropagation();
        void preview.play();
      }}
      onKeyDown={(e) => e.stopPropagation()}
    >
      <div className={classes.bars} aria-hidden>
        {voiceBars(style).map((height, i) => (
          <div key={i} className={classes.bar} style={{ height: `${height}%` }} />
        ))}
      </div>
      <div className={classes.centre}>
        <div className={classes.button}>
          {busy ? (
            <Loader size="sm" />
          ) : preview.playing ? (
            <IconPlayerStopFilled className={classes.icon} />
          ) : (
            <IconPlayerPlayFilled className={classes.icon} style={{ marginLeft: "6%" }} />
          )}
        </div>
      </div>
    </button>
  );
}
