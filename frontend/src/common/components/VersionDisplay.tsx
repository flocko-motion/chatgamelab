import { Text } from '@mantine/core';
import { useVersion } from '../../api/hooks';

interface VersionDisplayProps {
  darkMode?: boolean;
}

/** The release the server runs; frontend and server ship together. */
export function VersionDisplay({ darkMode = false }: VersionDisplayProps) {
  const { data } = useVersion();
  if (!data?.version) return null;

  return (
    <Text size="sm" c={darkMode ? 'gray.6' : 'dimmed'} span>
      {data.version === 'dev' ? data.version : `v${data.version}`}
    </Text>
  );
}
