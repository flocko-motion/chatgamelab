import { Text, Tooltip, Divider } from '@mantine/core';
import { useVersion } from '../../api/hooks';
import { version as frontendVersion, buildTime as frontendBuildTime } from '../../version';

interface BackendVersionInfo {
  version: string;
  buildTime: string;
  gitCommit: string;
}

export function VersionDisplay() {
  const { data: backendData, isLoading, isError } = useVersion();

  const backendInfo: BackendVersionInfo | null = 
    backendData && backendData.version && backendData.buildTime && backendData.gitCommit
      ? {
          version: backendData.version,
          buildTime: backendData.buildTime,
          gitCommit: backendData.gitCommit,
        }
      : null;

  const backendVersion = backendInfo?.version || (isError ? 'unknown' : 'loading...');
  const displayVersion = isLoading ? 'v{frontend} / loading...' : 
                         isError ? `v${frontendVersion} / unknown` :
                         `v${frontendVersion} / v${backendVersion}`;

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
          <Text size="sm" c="gray.2">
            v{backendInfo.version}
          </Text>
        </div>
        <div style={{ display: 'flex', gap: 'var(--mantine-spacing-sm)' }}>
          <Text size="sm" c="gray.2" style={{ minWidth: '60px' }}>
            <strong>Built:</strong>
          </Text>
          <Text size="sm" c="gray.2">
            {new Date(backendInfo.buildTime).toLocaleString()}
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
      <Text size="sm" c="dimmed" span style={{ cursor: 'help' }}>
        {displayVersion}
      </Text>
    </Tooltip>
  );
}
