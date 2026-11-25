import type { CSSProperties } from "react";
import { useBranding } from "@/providers/branding";
import symbol from "@/assets/logos/svg/ChatGameLab-Symbol-Colorful.svg";
import classes from "./SiteLogo.module.css";

/** A size in pixels, from the smallest screen up (Mantine's sm and lg). */
type Size = number | { base: number; sm?: number; lg?: number };

interface SiteLogoProps {
  /** The symbol's height; the name scales with it. */
  size: Size;
  /** "row": the name beside the symbol (header); "stacked": below it. */
  layout?: "row" | "stacked";
}

/**
 * The site's logo: ChatGameLab's symbol and the site's name as text, so an
 * instance that renames itself shows its own name.
 */
export function SiteLogo({ size, layout = "row" }: SiteLogoProps) {
  const { siteName } = useBranding();
  const s = typeof size === "number" ? { base: size } : size;
  const vars = {
    "--size-base": `${s.base}px`,
    ...(s.sm !== undefined && { "--size-sm": `${s.sm}px` }),
    ...(s.lg !== undefined && { "--size-lg": `${s.lg}px` }),
  } as CSSProperties;

  return (
    <span
      className={`${classes.root} ${layout === "stacked" ? classes.stacked : ""}`}
      style={vars}
      role="img"
      aria-label={siteName}
    >
      <img src={symbol} alt="" className={classes.symbol} />
      {siteName && (
        <span className={classes.name} aria-hidden="true">
          {siteName}
        </span>
      )}
    </span>
  );
}
