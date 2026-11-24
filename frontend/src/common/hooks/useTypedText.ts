import { useEffect, useState } from "react";

export interface TypedText {
  /** The whole text, without markup: what a screen reader reads, once. */
  full: string;
  /** The part typed so far. */
  typed: string;
  /** Where the marked word starts and ends in `full`; -1 when there is none. */
  markStart: number;
  markEnd: number;
}

const prefersReducedMotion = () =>
  typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

/**
 * Types a text out letter by letter. One word may be marked with
 * `<mark>…</mark>` (for a highlighter stroke); the markup is stripped and its
 * place returned. With "reduce motion" the whole text shows at once.
 */
export function useTypedText(source: string, msPerLetter = 38): TypedText {
  const open = source.indexOf("<mark>");
  const close = source.indexOf("</mark>");
  const marked = open >= 0 && close > open;
  const full = source.replace("<mark>", "").replace("</mark>", "");
  const markStart = marked ? open : -1;
  const markEnd = marked ? close - "<mark>".length : -1;

  const [reduce] = useState(prefersReducedMotion);
  const start = reduce ? full.length : 0;
  const [state, setState] = useState({ text: full, count: start });

  // A new text (another language, the name arriving) starts typing again.
  if (state.text !== full) setState({ text: full, count: start });

  useEffect(() => {
    if (reduce) return;
    const id = setInterval(
      () => setState((s) => (s.count >= s.text.length ? s : { ...s, count: s.count + 1 })),
      msPerLetter,
    );
    return () => clearInterval(id);
  }, [full, msPerLetter, reduce]);

  const count = state.text === full ? state.count : start;
  return { full, typed: full.slice(0, count), markStart, markEnd };
}
