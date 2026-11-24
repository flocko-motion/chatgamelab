import type { ReactNode } from "react";
import { UnstyledButton } from "@mantine/core";
import { IconX } from "@tabler/icons-react";
import classes from "./FilterBar.module.css";

export interface FilterBarOption {
  value: string;
  label: string;
  /** An icon before the label. */
  icon?: ReactNode;
  /** A small colour dot before the label (e.g. a genre's colour). */
  dot?: string;
}

export interface FilterBarGroup {
  options: FilterBarOption[];
  /** The values that are on. */
  active: readonly string[];
  onToggle: (value: string) => void;
}

export interface FilterBarProps {
  /** Groups of filters, set apart by a stronger line (e.g. who/where, genres). */
  groups: FilterBarGroup[];
  /** Turns every filter off; shown while one is on. */
  onClear: () => void;
  clearLabel: string;
  /** The bar's accessible name. */
  label: string;
}

/**
 * FilterBar - filters that each switch on and off and combine, as one bar
 *
 * USE WHEN:
 * - A list can be narrowed by several independent filters at once
 *   (Favorites + Adventure): no tabs, since these are no subpages
 *
 * One outlined bar, the filters its segments; every active one looks the
 * same (ink, white text). On a phone the bar scrolls sideways.
 */
export function FilterBar({ groups, onClear, clearLabel, label }: FilterBarProps) {
  const anyOn = groups.some((g) => g.active.length > 0);
  const shown = groups.filter((g) => g.options.length > 0);
  return (
    <div className={classes.row}>
      <div className={classes.bar} role="group" aria-label={label}>
        {shown.map((group, gi) => (
          <div key={gi} className={classes.group}>
            {group.options.map((option) => {
              const on = group.active.includes(option.value);
              return (
                <UnstyledButton
                  key={option.value}
                  className={classes.segment}
                  data-on={on || undefined}
                  aria-pressed={on}
                  onClick={() => group.onToggle(option.value)}
                >
                  {option.dot && <span className={classes.dot} style={{ background: option.dot }} aria-hidden />}
                  {option.icon}
                  {option.label}
                </UnstyledButton>
              );
            })}
          </div>
        ))}
      </div>
      {anyOn && (
        <UnstyledButton className={classes.clear} onClick={onClear}>
          <IconX size={14} aria-hidden />
          {clearLabel}
        </UnstyledButton>
      )}
    </div>
  );
}
