import { Box, Group } from "@mantine/core";
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
  const prevValuesRef = useRef<Record<string, string>>({});
  const [changedFields, setChangedFields] = useState<Set<string>>(new Set());

  // Detect value changes and trigger animation
  /* eslint-disable react-hooks/set-state-in-effect -- Intentional: update animation state when values change */
  useEffect(() => {
    const newChangedFields = new Set<string>();

    statusFields.forEach((field) => {
      const key = field.name || "";
      const prevValue = prevValuesRef.current[key];
      const currentValue = field.value || "";

      // If value changed (and we had a previous value), mark as changed
      if (prevValue !== undefined && prevValue !== currentValue) {
        newChangedFields.add(key);
      }

      // Update stored value
      prevValuesRef.current[key] = currentValue;
    });

    if (newChangedFields.size > 0) {
      setChangedFields(newChangedFields);

      // Clear animation after it completes
      const timer = setTimeout(() => {
        setChangedFields(new Set());
      }, 2000);

      return () => clearTimeout(timer);
    }
  }, [statusFields]);

  if (!statusFields || statusFields.length === 0) {
    return null;
  }

  return (
    <Box className={classes.statusBar} py="xs" style={cssVars}>
      <Group
        gap="sm"
        wrap="nowrap"
        px={{ base: "sm", sm: "md" }}
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
