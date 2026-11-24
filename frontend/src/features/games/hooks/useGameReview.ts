import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useReviewGame } from "@/api/hooks";
import { genreWording, pageOfField, type EditorPage } from "../lib/editorPages";
import type { GameFormFields } from "../lib/gameUpdate";

/** The coach's answer: one point, the section it is about (none: nothing to fix). */
export interface ReviewNote {
  page: EditorPage | null;
  text: string;
  /** The review could not be made: no key, a timeout, an error. */
  failed?: boolean;
}

/**
 * The editor's coach: a fast model looks at the form as it is and answers
 * with the one point most worth fixing, in the caller's language. An empty
 * required field is said in the editor's own words. A review that cannot
 * be made says so quietly; it never stands in the way of saving.
 */
export function useGameReview(values: GameFormFields) {
  const { t } = useTranslation("common");
  const review = useReviewGame();
  const [note, setNote] = useState<ReviewNote | null>(null);

  const ask = async () => {
    setNote(null);
    try {
      const point = await review.mutateAsync({
        genre: values.genre,
        name: values.name,
        description: values.description,
        systemMessageScenario: values.systemMessageScenario,
        systemMessageGameStart: values.systemMessageGameStart,
        imageStyle: values.imageStyle,
        voiceStyle: values.voiceStyle,
        statusFields: values.statusFields,
      });
      const page = pageOfField(point.field);
      if (point.missing) {
        const wording = genreWording(values.genre);
        setNote({ page, text: t(`games.editor.review.missing.${wording}.${point.field}`) });
      } else {
        setNote({ page, text: point.text || t("games.editor.review.looksGood") });
      }
    } catch {
      setNote({ page: null, text: t("games.editor.review.failed"), failed: true });
    }
  };

  return {
    ask,
    asking: review.isPending,
    note,
    dismiss: () => setNote(null),
  };
}
