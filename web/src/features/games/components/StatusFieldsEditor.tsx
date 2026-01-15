import {
  Stack,
  Group,
  TextInput,
  Textarea,
  SegmentedControl,
  ActionIcon,
  Text,
  Paper,
  Collapse,
} from '@mantine/core';
import { IconChevronDown, IconChevronRight, IconPlus } from '@tabler/icons-react';
import { TextButton, DeleteIconButton } from '@components/buttons';
import { useTranslation } from 'react-i18next';
import { useState, useMemo } from 'react';

interface SubField {
  name: string;
  value: string;
}

interface StatusField {
  name: string;
  value: string | SubField[];
}

interface StatusFieldsEditorProps {
  value: string;
  onChange: (value: string) => void;
}

type EditorMode = 'visual' | 'raw';

function isSubFieldArray(value: unknown): value is SubField[] {
  return Array.isArray(value) && value.every(
    (item) => typeof item === 'object' && item !== null && 
              typeof item.name === 'string' && typeof item.value === 'string'
  );
}

function parseStatusFields(jsonString: string): StatusField[] {
  if (!jsonString.trim()) return [];
  try {
    const parsed = JSON.parse(jsonString);
    if (Array.isArray(parsed)) {
      return parsed.filter(
        (item): item is StatusField =>
          typeof item === 'object' &&
          item !== null &&
          typeof item.name === 'string' &&
          (typeof item.value === 'string' || isSubFieldArray(item.value))
      );
    }
  } catch {
    // Invalid JSON, return empty
  }
  return [];
}

function stringifyStatusFields(fields: StatusField[]): string {
  if (fields.length === 0) return '';
  return JSON.stringify(fields);
}

export function StatusFieldsEditor({ value, onChange }: StatusFieldsEditorProps) {
  const { t } = useTranslation('common');
  const [mode, setMode] = useState<EditorMode>('visual');
  const [expandedFields, setExpandedFields] = useState<Set<number>>(new Set());
  
  // Parse fields from value - fully controlled by parent
  const fields = useMemo(() => parseStatusFields(value), [value]);
  
  // Check if value is valid JSON array
  const parseError = useMemo(() => {
    return value.trim() !== '' && parseStatusFields(value).length === 0;
  }, [value]);

  const handleModeChange = (newMode: string) => {
    if (newMode === 'visual' && parseError) {
      return;
    }
    setMode(newMode as EditorMode);
  };

  const handleFieldNameChange = (index: number, newName: string) => {
    const updated = [...fields];
    updated[index] = { ...updated[index], name: newName };
    onChange(stringifyStatusFields(updated));
  };

  const handleFieldValueChange = (index: number, newValue: string) => {
    const updated = [...fields];
    updated[index] = { ...updated[index], value: newValue };
    onChange(stringifyStatusFields(updated));
  };

  const handleAddField = () => {
    const updated = [...fields, { name: '', value: '' }];
    onChange(stringifyStatusFields(updated));
  };

  const handleRemoveField = (index: number) => {
    const updated = fields.filter((_, i) => i !== index);
    setExpandedFields((prev) => {
      const next = new Set(prev);
      next.delete(index);
      return next;
    });
    onChange(stringifyStatusFields(updated));
  };

  const handleConvertToSubFields = (index: number) => {
    const updated = [...fields];
    updated[index] = { ...updated[index], value: [{ name: '', value: '' }] };
    setExpandedFields((prev) => new Set(prev).add(index));
    onChange(stringifyStatusFields(updated));
  };

  const handleConvertToSimple = (index: number) => {
    const updated = [...fields];
    updated[index] = { ...updated[index], value: '' };
    setExpandedFields((prev) => {
      const next = new Set(prev);
      next.delete(index);
      return next;
    });
    onChange(stringifyStatusFields(updated));
  };

  const handleSubFieldChange = (fieldIndex: number, subIndex: number, key: 'name' | 'value', newValue: string) => {
    const updated = [...fields];
    const currentValue = updated[fieldIndex].value;
    if (Array.isArray(currentValue)) {
      const updatedSubs = [...currentValue];
      updatedSubs[subIndex] = { ...updatedSubs[subIndex], [key]: newValue };
      updated[fieldIndex] = { ...updated[fieldIndex], value: updatedSubs };
      onChange(stringifyStatusFields(updated));
    }
  };

  const handleAddSubField = (fieldIndex: number) => {
    const updated = [...fields];
    const currentValue = updated[fieldIndex].value;
    if (Array.isArray(currentValue)) {
      updated[fieldIndex] = { ...updated[fieldIndex], value: [...currentValue, { name: '', value: '' }] };
      onChange(stringifyStatusFields(updated));
    }
  };

  const handleRemoveSubField = (fieldIndex: number, subIndex: number) => {
    const updated = [...fields];
    const currentValue = updated[fieldIndex].value;
    if (Array.isArray(currentValue)) {
      const updatedSubs = currentValue.filter((_, i) => i !== subIndex);
      if (updatedSubs.length === 0) {
        updated[fieldIndex] = { ...updated[fieldIndex], value: '' };
        setExpandedFields((prev) => {
          const next = new Set(prev);
          next.delete(fieldIndex);
          return next;
        });
      } else {
        updated[fieldIndex] = { ...updated[fieldIndex], value: updatedSubs };
      }
      onChange(stringifyStatusFields(updated));
    }
  };

  const toggleExpanded = (index: number) => {
    setExpandedFields((prev) => {
      const next = new Set(prev);
      if (next.has(index)) {
        next.delete(index);
      } else {
        next.add(index);
      }
      return next;
    });
  };

  const handleRawChange = (newValue: string) => {
    onChange(newValue);
  };

  const hasSubFields = (field: StatusField): field is StatusField & { value: SubField[] } => {
    return Array.isArray(field.value);
  };

  return (
    <Stack gap="xs">
      <Group justify="space-between" align="center">
        <Text size="sm" fw={500}>
          {t('games.editFields.statusFields')}
        </Text>
        <SegmentedControl
          size="xs"
          value={mode}
          onChange={handleModeChange}
          data={[
            { label: t('games.statusFieldsEditor.visual'), value: 'visual' },
            { label: t('games.statusFieldsEditor.raw'), value: 'raw' },
          ]}
        />
      </Group>

      <Text size="xs" c="dimmed">
        {t('games.editFields.statusFieldsHint')}
      </Text>

      {mode === 'visual' ? (
        <Stack gap="xs">
          {fields.length === 0 ? (
            <Paper p="sm" withBorder>
              <Text size="sm" c="dimmed" ta="center">
                {t('games.statusFieldsEditor.noFields')}
              </Text>
            </Paper>
          ) : (
            fields.map((field, index) => (
              <Paper key={index} p="xs" withBorder>
                {hasSubFields(field) ? (
                  <Stack gap="xs">
                    <Group gap="xs" align="center" wrap="nowrap">
                      <ActionIcon
                        variant="subtle"
                        size="xs"
                        onClick={() => toggleExpanded(index)}
                      >
                        {expandedFields.has(index) ? <IconChevronDown size={14} /> : <IconChevronRight size={14} />}
                      </ActionIcon>
                      <TextInput
                        placeholder={t('games.statusFieldsEditor.namePlaceholder')}
                        value={field.name}
                        onChange={(e) => handleFieldNameChange(index, e.target.value)}
                        size="xs"
                        style={{ flex: 1 }}
                      />
                      <Text size="xs" c="dimmed">({field.value.length} {t('games.statusFieldsEditor.subFields')})</Text>
                      <DeleteIconButton
                        onClick={() => handleConvertToSimple(index)}
                        aria-label={t('games.statusFieldsEditor.convertToSimple')}
                      />
                      <DeleteIconButton
                        onClick={() => handleRemoveField(index)}
                        aria-label={t('delete')}
                      />
                    </Group>
                    <Collapse in={expandedFields.has(index)}>
                      <Stack gap="xs" ml="xl">
                        {field.value.map((subField, subIndex) => (
                          <Group key={subIndex} gap="xs" align="flex-end" wrap="nowrap">
                            <TextInput
                              placeholder={t('games.statusFieldsEditor.subNamePlaceholder')}
                              value={subField.name}
                              onChange={(e) => handleSubFieldChange(index, subIndex, 'name', e.target.value)}
                              size="xs"
                              style={{ flex: 1 }}
                            />
                            <TextInput
                              placeholder={t('games.statusFieldsEditor.valuePlaceholder')}
                              value={subField.value}
                              onChange={(e) => handleSubFieldChange(index, subIndex, 'value', e.target.value)}
                              size="xs"
                              style={{ flex: 1 }}
                            />
                            <DeleteIconButton
                              onClick={() => handleRemoveSubField(index, subIndex)}
                              aria-label={t('delete')}
                            />
                          </Group>
                        ))}
                        <TextButton
                          size="xs"
                          onClick={() => handleAddSubField(index)}
                          leftSection={<IconPlus size={14} />}
                        >
                          {t('games.statusFieldsEditor.addSubField')}
                        </TextButton>
                      </Stack>
                    </Collapse>
                  </Stack>
                ) : (
  <Group gap="xs" align="flex-end" wrap="nowrap">
                    <TextInput
                      placeholder={t('games.statusFieldsEditor.namePlaceholder')}
                      value={field.name}
                      onChange={(e) => handleFieldNameChange(index, e.target.value)}
                      size="xs"
                      style={{ flex: 1 }}
                    />
                    <TextInput
                      placeholder={t('games.statusFieldsEditor.valuePlaceholder')}
                      value={field.value as string}
                      onChange={(e) => handleFieldValueChange(index, e.target.value)}
                      size="xs"
                      style={{ flex: 1 }}
                    />
                    <TextButton
                      size="xs"
                      onClick={() => handleConvertToSubFields(index)}
                      leftSection={<IconPlus size={14} />}
                    >
                      {t('games.statusFieldsEditor.addSubField')}
                    </TextButton>
                    <DeleteIconButton
                      onClick={() => handleRemoveField(index)}
                      aria-label={t('delete')}
                    />
                  </Group>
                )}
              </Paper>
            ))
          )}
          <TextButton
            size="xs"
            onClick={handleAddField}
            leftSection={<IconPlus size={14} />}
          >
            {t('games.statusFieldsEditor.addField')}
          </TextButton>
        </Stack>
      ) : (
        <Textarea
          value={value}
          onChange={(e) => handleRawChange(e.target.value)}
          minRows={3}
          autosize
          maxRows={6}
          styles={{ input: { fontFamily: 'monospace' } }}
          error={parseError ? t('games.statusFieldsEditor.invalidJson') : undefined}
          placeholder='[{"name":"Health","value":"100"}]'
        />
      )}
    </Stack>
  );
}
