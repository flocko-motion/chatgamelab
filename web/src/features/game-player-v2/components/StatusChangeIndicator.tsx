import { Group } from '@mantine/core';
import { useTranslation } from 'react-i18next';
import type { ObjStatusField } from '@/api/generated';
import { useGameTheme } from '../theme';
import classes from './StatusChangeIndicator.module.css';

interface StatusChange {
  name: string;
  delta: number;
  emoji?: string;
}

interface StatusChangeIndicatorProps {
  currentFields: ObjStatusField[];
  previousFields: ObjStatusField[];
}

function parseNumericValue(value: string | undefined): number | null {
  if (!value) return null;
  // Extract numeric part from strings like "100", "100/100", "50%"
  const match = value.match(/^(-?\d+)/);
  return match ? parseInt(match[1], 10) : null;
}

function calculateChanges(
  current: ObjStatusField[],
  previous: ObjStatusField[],
  getEmoji: (name: string) => string
): StatusChange[] {
  const changes: StatusChange[] = [];
  
  const prevMap = new Map<string, string>();
  previous.forEach(field => {
    if (field.name) {
      prevMap.set(field.name, field.value || '');
    }
  });
  
  current.forEach(field => {
    if (!field.name) return;
    
    const currentValue = parseNumericValue(field.value);
    const prevValue = parseNumericValue(prevMap.get(field.name));
    
    if (currentValue !== null && prevValue !== null && currentValue !== prevValue) {
      const delta = currentValue - prevValue;
      changes.push({
        name: field.name,
        delta,
        emoji: getEmoji(field.name),
      });
    }
  });
  
  return changes;
}

export function StatusChangeIndicator({ currentFields, previousFields }: StatusChangeIndicatorProps) {
  const { t } = useTranslation('common');
  const { getStatusEmoji } = useGameTheme();
  
  const changes = calculateChanges(currentFields, previousFields, getStatusEmoji);
  
  if (changes.length === 0) {
    return null;
  }
  
  return (
    <div className={classes.container}>
      <span className={classes.label}>{t('gamePlayer.statusChange.effect')}</span>
      <Group gap="xs" wrap="wrap">
        {changes.map((change) => (
          <span 
            key={change.name} 
            className={`${classes.change} ${change.delta > 0 ? classes.positive : classes.negative}`}
          >
            {change.emoji && <span className={classes.emoji}>{change.emoji}</span>}
            <span className={classes.delta}>
              {change.delta > 0 ? '+' : ''}{change.delta}
            </span>
            <span className={classes.name}>{change.name}</span>
          </span>
        ))}
      </Group>
    </div>
  );
}
