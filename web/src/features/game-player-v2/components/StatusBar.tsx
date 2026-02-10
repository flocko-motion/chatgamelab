import { Box, Group, Popover, Stack } from "@mantine/core";
import { useMediaQuery } from "@mantine/hooks";
import { useEffect, useRef, useState } from "react";
import type { ObjStatusField } from "@/api/generated";
import { useGameTheme } from "../theme";
import { ThemedText } from "./text-effects";
import classes from "./GamePlayer.module.css";

interface StatusBarProps {
  statusFields: ObjStatusField[];
}

export function StatusBar({ statusFields }: StatusBarProps) {
  const { getStatusEmoji, cssVars } = useGameTheme();
  const isMobile = useMediaQuery("(max-width: 48em)");
  const prevValuesRef = useRef<Record<string, string>>({});
  const [changedFields, setChangedFields] = useState<Set<string>>(new Set());
  const [popoverOpened, setPopoverOpened] = useState(false);

  // Detect value changes and trigger animation
  /* eslint-disable react-hooks/set-state-in-effect -- Intentional: update animation state when values change */
  useEffect(() => {
    const newChangedFields = new Set<string>();

    statusFields.forEach((field) => {
      const key = field.name || "";
      const prevValue = prevValuesRef.current[key];
      const currentValue = field.value || "";

      if (prevValue !== undefined && prevValue !== currentValue) {
        newChangedFields.add(key);
      }

      prevValuesRef.current[key] = currentValue;
    });

    if (newChangedFields.size > 0) {
      setChangedFields(newChangedFields);

      const timer = setTimeout(() => {
        setChangedFields(new Set());
      }, 2000);

      return () => clearTimeout(timer);
    }
  }, [statusFields]);

  // Pick a "spotlight" field for mobile: recently changed field, or fall back to last field
  const recentField = statusFields.find((f) => changedFields.has(f.name || ""));
  const spotlightField = recentField || statusFields[statusFields.length - 1];
  const isSpotlightChanged = spotlightField
    ? changedFields.has(spotlightField.name || "")
    : false;

  if (!statusFields || statusFields.length === 0) {
    return null;
  }

  // ── Mobile: compact chip + 1 recently changed field ──────────────────
  if (isMobile) {
    return (
      <Box
        className={classes.statusBar}
        py="xs"
        style={{ ...cssVars, overflow: "hidden" }}
      >
        <Group gap="sm" wrap="nowrap" px="md">
          {/* Chip showing total field count — tap to expand */}
          <Popover
            opened={popoverOpened}
            onChange={setPopoverOpened}
            position="bottom"
            withArrow
            shadow="md"
            withinPortal
          >
            <Popover.Target>
              <div
                className={classes.statusOverflowChip}
                onClick={() => setPopoverOpened((o) => !o)}
              >
                <span className={classes.statusFieldValue}>▶ Stats</span>
              </div>
            </Popover.Target>
            <Popover.Dropdown
              p="xs"
              style={{
                ...cssVars,
                maxWidth: "90vw",
                background:
                  "linear-gradient(var(--game-bg-tint, transparent), var(--game-bg-tint, transparent)), var(--game-bg-status, var(--mantine-color-body))",
                border:
                  "1px solid var(--game-status-border, var(--mantine-color-default-border))",
              }}
            >
              <Stack gap="xs">
                {statusFields.map((field, index) => {
                  const emoji = field.name ? getStatusEmoji(field.name) : "";
                  const isChanged = changedFields.has(field.name || "");
                  return (
                    <div
                      key={field.name || index}
                      className={`${classes.statusField} ${isChanged ? classes.statusFieldChanged : ""}`}
                      style={{ whiteSpace: "normal", flexWrap: "wrap" }}
                    >
                      <span className={classes.statusFieldName}>
                        {emoji && (
                          <span className={classes.statusFieldEmoji}>
                            {emoji}
                          </span>
                        )}
                        {field.name}:
                      </span>
                      <span
                        className={`${classes.statusFieldValue} ${isChanged ? classes.statusFieldValueChanged : ""}`}
                      >
                        {field.value}
                      </span>
                    </div>
                  );
                })}
              </Stack>
            </Popover.Dropdown>
          </Popover>

          {/* Show 1 spotlight field next to the chip */}
          {spotlightField && (
            <div
              className={`${classes.statusField} ${isSpotlightChanged ? classes.statusFieldChanged : ""}`}
              style={{ overflow: "hidden" }}
            >
              <span className={classes.statusFieldName}>
                {spotlightField.name && (
                  <span className={classes.statusFieldEmoji}>
                    {getStatusEmoji(spotlightField.name)}
                  </span>
                )}
                <ThemedText
                  text={`${spotlightField.name}:`}
                  scope="statusFields"
                />
              </span>
              <span
                className={`${classes.statusFieldValue} ${isSpotlightChanged ? classes.statusFieldValueChanged : ""}`}
                style={{ overflow: "hidden", textOverflow: "ellipsis" }}
              >
                <ThemedText
                  text={spotlightField.value || ""}
                  scope="statusFields"
                />
              </span>
            </div>
          )}
        </Group>
      </Box>
    );
  }

  // ── Desktop: all fields in a scrollable row ──────────────────────────
  return (
    <Box className={classes.statusBar} py="xs" style={cssVars}>
      <Group
        gap="sm"
        wrap="nowrap"
        px="md"
        style={{
          width: "fit-content",
          minWidth: "100%",
          justifyContent: "center",
        }}
      >
        {statusFields.map((field, index) => {
          const emoji = field.name ? getStatusEmoji(field.name) : "";
          const key = field.name || String(index);
          const isChanged = changedFields.has(field.name || "");

          return (
            <div
              key={key}
              className={`${classes.statusField} ${isChanged ? classes.statusFieldChanged : ""}`}
            >
              <span className={classes.statusFieldName}>
                {emoji && (
                  <span className={classes.statusFieldEmoji}>{emoji}</span>
                )}
                <ThemedText text={`${field.name}:`} scope="statusFields" />
              </span>
              <span
                className={`${classes.statusFieldValue} ${isChanged ? classes.statusFieldValueChanged : ""}`}
              >
                <ThemedText text={field.value || ""} scope="statusFields" />
              </span>
            </div>
          );
        })}
      </Group>
    </Box>
  );
}
