import { useState, useRef, useEffect } from 'react';
import { TextInput, ActionIcon, Transition } from '@mantine/core';
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
    <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
      <Transition mounted={expanded} transition="scale-x" duration={150}>
        {(styles) => (
          <TextInput
            ref={inputRef}
            value={value}
            onChange={(e) => onChange(e.currentTarget.value)}
            placeholder={placeholder}
            size="xs"
            style={{ ...styles, width: 150 }}
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
      </Transition>
      <ActionIcon
        variant={expanded || value ? 'light' : 'subtle'}
        color={value ? 'violet' : 'gray'}
        onClick={handleToggle}
        aria-label={placeholder || 'Search'}
      >
        <IconSearch size={18} />
      </ActionIcon>
    </div>
  );
}
