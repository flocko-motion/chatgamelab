import { ScrollArea, Stack } from "@mantine/core";
import { BodyText } from "@components/typography";

/** The title (first line, bold in the Markdown) and the body of a declaration. */
function splitDeclaration(text: string): { title: string; body: string } {
  const [first = "", ...rest] = text.split("\n");
  return { title: first.replace(/^\*\*(.*)\*\*$/, "$1"), body: rest.join("\n").trim() };
}

/**
 * The binding declaration in a scrollable box, in German whatever the
 * interface's language; its wording is shown untouched.
 */
export function DeclarationText({ text }: { text: string }) {
  const { title, body } = splitDeclaration(text);
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
      <Stack gap="xs">
        <BodyText size="sm" fw={700}>
          {title}
        </BodyText>
        <div style={{ whiteSpace: "pre-wrap" }}>
          <BodyText size="sm">{body}</BodyText>
        </div>
      </Stack>
    </ScrollArea.Autosize>
  );
}
