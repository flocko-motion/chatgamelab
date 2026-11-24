import type { ReactNode } from 'react';
import classes from './typography.module.css';

export interface ItemTitleProps {
  children: ReactNode;
  /** Lines before it is cut with an ellipsis (default 2). */
  lines?: 1 | 2;
}

/**
 * ItemTitle - the name of one item in a list or grid (a game in a tile, a row)
 *
 * USE WHEN:
 * - Naming a thing among many: smaller than a CardTitle, bold, cut after
 *   one or two lines
 *
 * @example
 * <ItemTitle>{game.name}</ItemTitle>
 */
export function ItemTitle({ children, lines = 2 }: ItemTitleProps) {
  return (
    <h3 className={classes.itemTitle} data-lines={lines}>
      {children}
    </h3>
  );
}
