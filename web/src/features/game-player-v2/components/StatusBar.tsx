import { Box, Group } from '@mantine/core';
import type { ObjStatusField } from '@/api/generated';
import { useGameTheme } from '../theme';
import classes from './GamePlayer.module.css';

interface StatusBarProps {
  statusFields: ObjStatusField[];
}

export function StatusBar({ statusFields }: StatusBarProps) {
  const { getStatusEmoji, cssVars } = useGameTheme();
  
  if (!statusFields || statusFields.length === 0) {
    return null;
  }

  return (
    <Box className={classes.statusBar} px={{ base: 'sm', sm: 'md' }} py="xs" style={cssVars}>
      <Group gap="sm" wrap="nowrap">
        {statusFields.map((field, index) => {
          const emoji = field.name ? getStatusEmoji(field.name) : '';
          return (
            <div key={field.name || index} className={classes.statusField}>
              <span className={classes.statusFieldName}>
                {emoji && <span className={classes.statusFieldEmoji}>{emoji}</span>}
                {field.name}:
              </span>
              <span className={classes.statusFieldValue}>{field.value}</span>
            </div>
          );
        })}
      </Group>
    </Box>
  );
}
