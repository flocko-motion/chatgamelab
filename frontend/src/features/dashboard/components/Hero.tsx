import { useState } from "react";
import { IconArrowRight, IconSparkles } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { ActionButton } from "@components/buttons";
import { Kicker } from "@components/typography";
import { useTypedText } from "@/common/hooks/useTypedText";
import classes from "./Dashboard.module.css";

export interface HeroProps {
  /** The user's first name; empty while unknown. */
  name: string;
  /** Opens the editor for a new game, "what is it about" filled in. */
  onCreate: (about: string) => void;
}

/**
 * The dashboard's top: a typed greeting that asks what to build, and the
 * prompt that starts a new game from an idea.
 */
export function Hero({ name, onCreate }: HeroProps) {
  const { t, i18n } = useTranslation("dashboard");
  const [idea, setIdea] = useState("");
  const greeting = useTypedText(
    name ? t("hero.greeting", { name, interpolation: { escapeValue: false } }) : t("hero.greetingNoName"),
  );
  const today = new Intl.DateTimeFormat(i18n.language, {
    weekday: "long",
    day: "numeric",
    month: "long",
  }).format(new Date());

  const { typed, markStart, markEnd } = greeting;
  const before = markStart < 0 ? typed : typed.slice(0, markStart);
  const mark = markStart < 0 ? "" : typed.slice(markStart, markEnd);
  const after = markStart < 0 ? "" : typed.slice(markEnd);

  return (
    <section className={classes.hero}>
      <Kicker hand>{t("hero.journal", { date: today })}</Kicker>
      <h1 className={classes.greeting} aria-label={greeting.full}>
        <span aria-hidden>
          {before}
          {mark && <span className={classes.marker}>{mark}</span>}
          {after}
          <span className={classes.caret} />
        </span>
      </h1>
      <form
        className={classes.prompt}
        onSubmit={(event) => {
          event.preventDefault();
          onCreate(idea);
        }}
      >
        <IconSparkles size={20} className={classes.promptIcon} aria-hidden />
        <input
          className={classes.promptInput}
          value={idea}
          onChange={(event) => setIdea(event.target.value)}
          aria-label={t("hero.promptLabel")}
          placeholder={t("hero.promptPlaceholder")}
        />
        <ActionButton type="submit" size="md" rightSection={<IconArrowRight size={16} />}>
          {t("hero.create")}
        </ActionButton>
      </form>
    </section>
  );
}
