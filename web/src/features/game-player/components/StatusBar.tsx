import { Box, Group } from '@mantine/core';
import type { ObjStatusField } from '@/api/generated';
import classes from './GamePlayer.module.css';

interface StatusBarProps {
  statusFields: ObjStatusField[];
}

export function StatusBar({ statusFields }: StatusBarProps) {
  if (!statusFields || statusFields.length === 0) {
    return null;
  }

  return (
    <Box className={classes.statusBar} px={{ base: 'sm', sm: 'md' }} py="xs">
      <Group gap="sm" wrap="nowrap">
        {statusFields.map((field, index) => (
          <div key={field.name || index} className={classes.statusField}>
            <span className={classes.statusFieldName}>{field.name}:</span>
            <span className={classes.statusFieldValue}>{field.value}</span>
          </div>
        ))}
      </Group>
    </Box>
  );
}
