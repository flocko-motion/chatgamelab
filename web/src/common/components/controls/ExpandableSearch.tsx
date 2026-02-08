import { useState, useRef, useEffect } from 'react';
import { TextInput, ActionIcon } from '@mantine/core';
import { IconSearch, IconX } from '@tabler/icons-react';

export interface ExpandableSearchProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
}

export function ExpandableSearch({ value, onChange, placeholder }: ExpandableSearchProps) {
  const [expanded, setExpanded] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (expanded && inputRef.current) {
      inputRef.current.focus();
    }
  }, [expanded]);

  const handleToggle = () => {
    if (expanded && value) {
      onChange('');
    }
    setExpanded(!expanded);
  };

  const handleBlur = () => {
    if (!value) {
      setExpanded(false);
    }
  };

  return (
    <div style={{
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'flex-end',
      gap: 4,
      flex: 1,
      minWidth: 0,
    }}>
      {expanded && (
        <TextInput
          ref={inputRef}
          value={value}
          onChange={(e) => onChange(e.currentTarget.value)}
          placeholder={placeholder}
          size="xs"
          style={{ flex: 1, minWidth: 0 }}
          onBlur={handleBlur}
          rightSection={
            value ? (
              <ActionIcon size="xs" variant="subtle" onClick={() => onChange('')}>
                <IconX size={14} />
              </ActionIcon>
            ) : null
          }
        />
      )}
      <ActionIcon
        variant={expanded || value ? 'light' : 'subtle'}
        color={value ? 'accent' : 'gray'}
        onClick={handleToggle}
        aria-label={placeholder || 'Search'}
        style={{ flexShrink: 0 }}
      >
        <IconSearch size={18} />
      </ActionIcon>
    </div>
  );
}
