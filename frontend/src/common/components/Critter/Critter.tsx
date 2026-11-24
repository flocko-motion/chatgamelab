import { useId } from 'react';
import { critterParts } from '@/common/lib/critter';
import classes from './Critter.module.css';

export interface CritterProps {
  /** What the critter is drawn from: the user's id (stable), else their name. */
  seed: string;
  /** Width and height in px. */
  size?: number;
  /** A white gap and a ring in the critter's colour around it. */
  ring?: boolean;
}

// Bodies in a 40×40 box, roughly centred at (20, 23).
const BODIES = [
  'M20 8c8 0 13 6 13 14s-4 12-13 12S7 30 7 22 12 8 20 8z',
  'M9 14c3-5 19-5 22 0s4 15-2 18-15 3-19 0S6 19 9 14z',
  'M20 7c5 0 7 4 10 6s4 9 2 14-6 7-12 7-10-2-12-7-1-11 2-14 5-6 10-6z',
  'M8 33V20a12 12 0 0124 0v13l-4-3-4 3-4-3-4 3-4-3z',
];

const INK = 'var(--mantine-color-ink-5)';
const WHITE = 'var(--mantine-color-white)';

/**
 * Critter - a user's avatar: a little lab creature drawn from a seed
 *
 * USE WHEN:
 * - Showing who someone is next to their name (header, game cards, lists)
 *
 * The same seed always draws the same critter (critterParts); it is drawn in
 * the browser, nothing is stored or fetched. Decorative: the name next to it
 * says who it is, so it is hidden from screen readers.
 *
 * @example
 * <Critter seed={user.id} size={28} />
 */
export function Critter({ seed, size = 28, ring = false }: CritterProps) {
  const gradientId = useId();
  const p = critterParts(seed);
  const { light, main, bg } = p.palette;

  const eye = (x: number, y: number, r: number) => (
    <g key={`${x}-${y}`}>
      <circle cx={x} cy={y} r={r} fill={WHITE} />
      <circle cx={x + p.look} cy={y + 0.4} r={r * 0.55} fill={INK} />
      <circle cx={x + p.look - r * 0.2} cy={y - r * 0.15} r={r * 0.18} fill={WHITE} />
    </g>
  );
  const eyes = [
    [eye(20, 19, 4.2)],
    [eye(15.5, 19, 3.2), eye(24.5, 19, 3.2)],
    [eye(13.5, 20, 2.5), eye(20, 16.5, 2.8), eye(26.5, 20, 2.5)],
  ][p.eyes];
  const mouth = [
    <path key="m" d="M16 26q4 4 8 0" stroke={INK} strokeWidth="1.6" fill="none" strokeLinecap="round" />,
    <ellipse key="m" cx="20" cy="27" rx="2" ry="2.4" fill={INK} />,
    <path key="m" d="M16.5 27h7" stroke={INK} strokeWidth="1.6" strokeLinecap="round" />,
    <g key="m">
      <path d="M15.5 25.5q4.5 5 9 0z" fill={INK} />
      <rect x="18.6" y="25.5" width="2.8" height="1.6" fill={WHITE} />
    </g>,
  ][p.mouth];
  const topper = [
    null,
    <g key="t">
      <path d="M20 8V3" stroke={main} strokeWidth="1.6" strokeLinecap="round" />
      <circle cx="20" cy="3" r="2" fill="var(--mantine-color-highlight-5)" />
    </g>,
    <g key="t">
      <path d="M12 12l-3-6 6 3z" fill={main} />
      <path d="M28 12l3-6-6 3z" fill={main} />
    </g>,
    <g key="t">
      <path d="M20 8c0-3 1-5 4-6-1 3-2 4-4 6z" fill="var(--mantine-color-green-4)" />
      <path d="M20 8c0-2-1-4-3-5 0 2 1 4 3 5z" fill="var(--mantine-color-green-6)" />
    </g>,
    <g key="t" fill={WHITE} opacity=".9">
      <circle cx="11" cy="9" r="1.6" />
      <circle cx="29" cy="7" r="1.2" />
      <circle cx="31" cy="11" r=".8" />
    </g>,
  ][p.topper];

  return (
    <svg
      className={classes.critter}
      data-ring={ring || undefined}
      style={{ '--critter-ring': main } as React.CSSProperties}
      width={size}
      height={size}
      viewBox="0 0 40 40"
      aria-hidden
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor={light} />
          <stop offset="1" stopColor={main} />
        </linearGradient>
      </defs>
      <circle cx="20" cy="20" r="20" fill={bg} />
      {topper}
      <path d={BODIES[p.body]} fill={`url(#${gradientId})`} />
      <ellipse cx="14" cy="27" rx="2.2" ry="1.3" fill="var(--mantine-color-highlight-5)" opacity=".35" />
      <ellipse cx="26" cy="27" rx="2.2" ry="1.3" fill="var(--mantine-color-highlight-5)" opacity=".35" />
      {eyes}
      {mouth}
    </svg>
  );
}
