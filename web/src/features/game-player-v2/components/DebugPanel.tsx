import { Box, Text, Code, Collapse, Group, Badge } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { IconChevronDown, IconChevronRight } from '@tabler/icons-react';
import { useTranslation } from 'react-i18next';
import type { SceneMessage } from '../types';
import classes from './DebugPanel.module.css';

interface DebugPanelProps {
  message: SceneMessage;
}

interface DebugFieldProps {
  label: string;
  hint: string;
  value: string | number | boolean | object | undefined | null;
  type?: 'string' | 'number' | 'boolean' | 'json';
}

function DebugField({ label, hint, value, type = 'string' }: DebugFieldProps) {
  if (value === undefined || value === null || value === '') return null;
  
  const displayValue = type === 'json' 
    ? JSON.stringify(value, null, 2)
    : String(value);
  
  return (
    <Box className={classes.field}>
      <Group gap="xs" wrap="nowrap">
        <Text size="xs" fw={600} className={classes.fieldLabel}>{label}</Text>
        <Text size="xs" c="dimmed" className={classes.fieldHint}>({hint})</Text>
      </Group>
      <Code block={type === 'json'} className={classes.fieldValue}>
        {displayValue}
      </Code>
    </Box>
  );
}

export function DebugPanel({ message }: DebugPanelProps) {
  const { t } = useTranslation('common');
  const [opened, { toggle }] = useDisclosure(false);
  
  const hasDebugData = message.statusFields?.length || message.imagePrompt || message.seq;
  
  if (!hasDebugData && message.type === 'player') {
    return null;
  }

  return (
    <Box className={classes.debugPanel}>
      <Group 
        gap="xs" 
        className={classes.header}
        onClick={toggle}
        style={{ cursor: 'pointer' }}
      >
        {opened ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
        <Text size="xs" fw={600}>{t('gamePlayer.debug.title')}</Text>
        <Badge size="xs" variant="light" color="gray">
          {message.type}
        </Badge>
        {message.seq && (
          <Badge size="xs" variant="outline" color="gray">
            #{message.seq}
          </Badge>
        )}
      </Group>
      
      <Collapse in={opened}>
        <Box className={classes.content} mt="xs">
          <DebugField 
            label={t('gamePlayer.debug.fields.id')}
            hint={t('gamePlayer.debug.hints.id')}
            value={message.id}
          />
          
          <DebugField 
            label={t('gamePlayer.debug.fields.type')}
            hint={t('gamePlayer.debug.hints.type')}
            value={message.type}
          />
          
          <DebugField 
            label={t('gamePlayer.debug.fields.seq')}
            hint={t('gamePlayer.debug.hints.seq')}
            value={message.seq}
            type="number"
          />
          
          {message.imagePrompt && (
            <DebugField 
              label={t('gamePlayer.debug.fields.imagePrompt')}
              hint={t('gamePlayer.debug.hints.imagePrompt')}
              value={message.imagePrompt}
            />
          )}
          
          {message.statusFields && message.statusFields.length > 0 && (
            <DebugField 
              label={t('gamePlayer.debug.fields.statusFields')}
              hint={t('gamePlayer.debug.hints.statusFields')}
              value={message.statusFields}
              type="json"
            />
          )}
          
          <DebugField 
            label={t('gamePlayer.debug.fields.isStreaming')}
            hint={t('gamePlayer.debug.hints.isStreaming')}
            value={message.isStreaming}
            type="boolean"
          />
          
          <DebugField 
            label={t('gamePlayer.debug.fields.timestamp')}
            hint={t('gamePlayer.debug.hints.timestamp')}
            value={message.timestamp?.toISOString()}
          />
        </Box>
      </Collapse>
    </Box>
  );
}
