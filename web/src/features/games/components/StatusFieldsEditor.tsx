import { Stack, Group, TextInput, Text, Paper } from "@mantine/core";
import { IconPlus } from "@tabler/icons-react";
import { TextButton, DeleteIconButton } from "@components/buttons";
import { useTranslation } from "react-i18next";
import { useMemo } from "react";

interface StatusField {
  name: string;
  value: string;
}

interface StatusFieldsEditorProps {
  value: string;
  onChange: (value: string) => void;
  readOnly?: boolean;
}

function parseStatusFields(jsonString: string): StatusField[] {
  if (!jsonString.trim()) return [];
  try {
    const parsed = JSON.parse(jsonString);
    if (Array.isArray(parsed)) {
      return parsed
        .filter(
          (item): item is StatusField =>
            typeof item === "object" &&
            item !== null &&
            typeof item.name === "string" &&
            typeof item.value === "string",
        )
        .map((item) => ({
          name: item.name,
          value:
            typeof item.value === "string" ? item.value : String(item.value),
        }));
    }
  } catch {
    // Invalid JSON, return empty
  }
  return [];
}

function stringifyStatusFields(fields: StatusField[]): string {
  if (fields.length === 0) return "";
  return JSON.stringify(fields);
}

export function StatusFieldsEditor({
  value,
  onChange,
  readOnly = false,
}: StatusFieldsEditorProps) {
  const { t } = useTranslation("common");

  // Parse fields from value - fully controlled by parent
  const fields = useMemo(() => parseStatusFields(value), [value]);

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
    const updated = [...fields, { name: "", value: "" }];
    onChange(stringifyStatusFields(updated));
  };

  const handleRemoveField = (index: number) => {
    const updated = fields.filter((_, i) => i !== index);
    onChange(stringifyStatusFields(updated));
  };

  return (
    <Stack gap="xs">
      <Text size="sm" fw={500}>
        {t("games.editFields.statusFields")}
      </Text>

      <Text size="xs" c="dimmed">
        {t("games.editFields.statusFieldsHint")}
      </Text>

      <Stack gap="xs">
        {fields.length === 0 ? (
          <Paper p="sm" withBorder>
            <Text size="sm" c="dimmed" ta="center">
              {t("games.statusFieldsEditor.noFields")}
            </Text>
          </Paper>
        ) : (
          fields.map((field, index) => (
            <Paper key={index} p="xs" withBorder>
              <Group gap="xs" align="flex-end" wrap="nowrap">
                <TextInput
                  placeholder={t("games.statusFieldsEditor.namePlaceholder")}
                  value={field.name}
                  onChange={(e) => handleFieldNameChange(index, e.target.value)}
                  style={{ flex: 1 }}
                  readOnly={readOnly}
                />
                <TextInput
                  placeholder={t("games.statusFieldsEditor.valuePlaceholder")}
                  value={field.value}
                  onChange={(e) =>
                    handleFieldValueChange(index, e.target.value)
                  }
                  style={{ flex: 1 }}
                  readOnly={readOnly}
                />
                {!readOnly && (
                  <DeleteIconButton
                    onClick={() => handleRemoveField(index)}
                    aria-label={t("delete")}
                  />
                )}
              </Group>
            </Paper>
          ))
        )}
        {!readOnly && (
          <TextButton
            size="xs"
            onClick={handleAddField}
            leftSection={<IconPlus size={14} />}
          >
            {t("games.statusFieldsEditor.addField")}
          </TextButton>
        )}
      </Stack>
    </Stack>
  );
}
