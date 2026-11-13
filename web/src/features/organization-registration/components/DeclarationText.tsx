import { ScrollArea, Text } from "@mantine/core";

/**
 * Shows the binding declaration text in a scrollable box.
 * The file is Markdown; only the bold markers of the title are rendered, the wording stays untouched.
 */
export function DeclarationText({ text }: { text: string }) {
  const lines = text.split("\n");
  const [first, ...rest] = lines;
  const title = first.replace(/^\*\*(.*)\*\*$/, "$1");

  return (
    <ScrollArea.Autosize
      mah={260}
      type="always"
      offsetScrollbars
      style={{
        border: "1px solid var(--mantine-color-gray-4)",
        borderRadius: "var(--mantine-radius-sm)",
      }}
      p="sm"
      lang="de"
      aria-label={title}
      tabIndex={0}
    >
      <Text fw={700} size="sm" mb="xs">
        {title}
      </Text>
      <Text size="sm" style={{ whiteSpace: "pre-wrap" }}>
        {rest.join("\n").trim()}
      </Text>
    </ScrollArea.Autosize>
  );
}
