import type { ReactNode } from 'react';
import classes from './typography.module.css';

export interface KickerProps {
  children: ReactNode;
  /** Handwritten, in magenta: decoration only, such as a date line. */
  hand?: boolean;
}

/**
 * Kicker - a short label above or beside a heading
 *
 * USE WHEN:
 * - Adding a small hint to a title ("pick up where you left off"), or a
 *   decorative line above a greeting (hand)
 *
 * Small mono capitals by default; never the only place of information a user
 * needs to act on.
 *
 * @example
 * <Kicker>pick up where you left off</Kicker>
 */
export function Kicker({ children, hand = false }: KickerProps) {
  return (
    <span className={classes.kicker} data-hand={hand || undefined}>
      {children}
    </span>
  );
}
