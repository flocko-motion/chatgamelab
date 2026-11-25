import type { ReactNode } from "react";
import { UnstyledButton } from "@mantine/core";
import { IconCheck, IconX } from "@tabler/icons-react";
import classes from "./FilterBar.module.css";

export interface FilterBarOption {
  value: string;
  label: string;
  /** An icon before the label. */
  icon?: ReactNode;
  /** A small colour dot before the label (e.g. a genre's colour); switched
   *  on, the segment takes this colour. */
  dot?: string;
}

export interface FilterBarGroup {
  options: FilterBarOption[];
  /** The values that are on. */
  active: readonly string[];
  onToggle: (value: string) => void;
}

export interface FilterBarProps {
  /** Groups of filters, each its own bar (e.g. who/where, genres). */
  groups: FilterBarGroup[];
  /** Brings every filter back to how the list starts; shown while one differs. */
  onClear: () => void;
  /** Whether a filter differs from how the list starts (default: one is on). */
  changed?: boolean;
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
 * Each group is an outlined bar; the filters are its segments. A filter that is on is coloured (its dot's colour, else cyan)
 * and ticked, so on and off tell apart at a glance. On a phone the row
 * scrolls sideways.
 */
export function FilterBar({ groups, onClear, changed, clearLabel, label }: FilterBarProps) {
  const anyOn = changed ?? groups.some((g) => g.active.length > 0);
  const shown = groups.filter((g) => g.options.length > 0);
  return (
    <div className={classes.row} role="group" aria-label={label}>
      {shown.map((group, gi) => (
        <div key={gi} className={classes.group}>
          <div className={classes.bar}>
            {group.options.map((option) => {
              const on = group.active.includes(option.value);
              return (
                <UnstyledButton
                  key={option.value}
                  className={classes.segment}
                  data-on={on || undefined}
                  aria-pressed={on}
                  onClick={() => group.onToggle(option.value)}
                  style={on && option.dot ? { background: option.dot } : undefined}
                >
                  {on && <IconCheck size={14} stroke={3} aria-hidden />}
                  {option.dot && !on && (
                    <span className={classes.dot} style={{ background: option.dot }} aria-hidden />
                  )}
                  {option.icon}
                  {option.label}
                </UnstyledButton>
              );
            })}
          </div>
        </div>
      ))}
      {anyOn && (
        <UnstyledButton className={classes.clear} onClick={onClear}>
          <IconX size={14} aria-hidden />
          {clearLabel}
        </UnstyledButton>
      )}
    </div>
  );
}
