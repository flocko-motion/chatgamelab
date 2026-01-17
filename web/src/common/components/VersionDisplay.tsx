import { Text, Tooltip, Divider, Alert, Group } from '@mantine/core';
import { IconAlertTriangle } from '@tabler/icons-react';
import { useTranslation, Trans } from 'react-i18next';
import { useVersion } from '../../api/hooks';
import { version as frontendVersion, buildTime as frontendBuildTime } from '../../version';

interface BackendVersionInfo {
  version: string;
  buildTime: string;
  gitCommit: string;
}

interface VersionDisplayProps {
  darkMode?: boolean;
}

export function VersionDisplay({ darkMode = false }: VersionDisplayProps) {
  const { t } = useTranslation('dashboard');
  const { data: backendData, isError } = useVersion();

  const backendInfo: BackendVersionInfo | null = 
    backendData && backendData.version && backendData.buildTime && backendData.gitCommit
      ? {
          version: backendData.version,
          buildTime: backendData.buildTime,
          gitCommit: backendData.gitCommit,
        }
      : null;

  // Check for version mismatch
  const hasVersionMismatch = backendInfo && backendInfo.version !== frontendVersion;
  
  const frontendLabel = (
    <div style={{ marginBottom: 'var(--mantine-spacing-sm)' }}>
      <Text size="lg" c="white" fw={600} mb="xs">
        Web
      </Text>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--mantine-spacing-xs)' }}>
        <div style={{ display: 'flex', gap: 'var(--mantine-spacing-sm)' }}>
          <Text size="sm" c="gray.2" style={{ minWidth: '60px' }}>
            <strong>Version:</strong>
          </Text>
          <Text size="sm" c="gray.2">
            v{frontendVersion}
          </Text>
        </div>
        <div style={{ display: 'flex', gap: 'var(--mantine-spacing-sm)' }}>
          <Text size="sm" c="gray.2" style={{ minWidth: '60px' }}>
            <strong>Built:</strong>
          </Text>
          <Text size="sm" c="gray.2">
            {new Date(frontendBuildTime).toLocaleString()}
          </Text>
        </div>
      </div>
    </div>
  );

  const backendLabel = backendInfo ? (
    <div>
      <Text size="lg" c="white" fw={600} mb="xs">
        Backend
      </Text>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--mantine-spacing-xs)' }}>
        <div style={{ display: 'flex', gap: 'var(--mantine-spacing-sm)' }}>
          <Text size="sm" c="gray.2" style={{ minWidth: '60px' }}>
            <strong>Version:</strong>
          </Text>
          <Text size="sm" c={hasVersionMismatch ? 'red.4' : 'gray.2'}>
            v{backendInfo.version}
          </Text>
        </div>
        <div style={{ display: 'flex', gap: 'var(--mantine-spacing-sm)' }}>
          <Text size="sm" c="gray.2" style={{ minWidth: '60px' }}>
            <strong>Built:</strong>
          </Text>
          <Text size="sm" c="gray.2">
            {backendInfo.buildTime === 'unknown' ? 'Unknown' : new Date(backendInfo.buildTime).toLocaleString()}
          </Text>
        </div>
        <div style={{ display: 'flex', gap: 'var(--mantine-spacing-sm)' }}>
          <Text size="sm" c="gray.2" style={{ minWidth: '60px' }}>
            <strong>Commit:</strong>
          </Text>
          <Text size="sm" c="blue.3" style={{ fontFamily: 'monospace' }}>
            {backendInfo.gitCommit.substring(0, 7)}
          </Text>
        </div>
      </div>
    </div>
  ) : (
    <div>
      <Text size="lg" c="white" fw={600} mb="xs">
        Backend
      </Text>
      <Text size="sm" c="gray.2">
        {isError ? 'Unknown' : 'Loading...'}
      </Text>
    </div>
  );

  const versionMismatchAlert = hasVersionMismatch ? (
    <Alert 
      variant="outline" 
      color="red" 
      title={t('version.mismatch.title')}
      icon={<IconAlertTriangle size={16} />}
      style={{ marginTop: 'var(--mantine-spacing-sm)' }}
    >
      <Text size="sm" c="red">
        <Trans 
        i18nKey="version.mismatch.message" 
        t={t}
        values={{
          frontendVersion: `v${frontendVersion}`,
          backendVersion: `v${backendInfo?.version}`
        }}
        components={{
          frontendVersion: <Text span c="blue.3" fw={600} />,
          backendVersion: <Text span c="orange.3" fw={600} />
        }}
      />
      </Text>
    </Alert>
  ) : null;

  return (
    <Tooltip 
      label={
        <div style={{ 
          display: 'flex', 
          flexDirection: 'column', 
          gap: 'var(--mantine-spacing-md)',
          padding: 'var(--mantine-spacing-sm)',
          minWidth: '280px'
        }}>
          {frontendLabel}
          <Divider color="gray.7" />
          {backendLabel}
          {versionMismatchAlert}
        </div>
      }
      withArrow
      position="top"
      multiline
      bg="dark.8"
      c="white"
      radius="md"
      maw={400}
    >
      <Group gap="xs" style={{ cursor: 'help', alignItems: 'center' }}>
        <Text size="sm" c={darkMode ? 'gray.6' : 'dimmed'} span>
          v{frontendVersion}
        </Text>
        {hasVersionMismatch && (
          <IconAlertTriangle 
            size={14} 
            color="red" 
            style={{ 
              animation: 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite' 
            }} 
          />
        )}
      </Group>
    </Tooltip>
  );
}
