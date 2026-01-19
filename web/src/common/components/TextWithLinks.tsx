import { Anchor, Text, type TextProps } from '@mantine/core';
import { Fragment, useMemo } from 'react';

interface TextWithLinksProps extends Omit<TextProps, 'children'> {
  children: string;
}

/**
 * Text component that automatically converts URLs to clickable links.
 * URLs are opened in a new tab with security attributes.
 */
export function TextWithLinks({ children, ...textProps }: TextWithLinksProps) {
  const parts = useMemo(() => {
    // Match URLs starting with http:// or https://
    const urlRegex = /(https?:\/\/[^\s]+)/g;
    const segments: Array<{ type: 'text' | 'link'; content: string }> = [];
    let lastIndex = 0;
    let match;

    while ((match = urlRegex.exec(children)) !== null) {
      // Add text before the URL
      if (match.index > lastIndex) {
        segments.push({ type: 'text', content: children.slice(lastIndex, match.index) });
      }
      // Add the URL
      segments.push({ type: 'link', content: match[0] });
      lastIndex = match.index + match[0].length;
    }

    // Add remaining text after last URL
    if (lastIndex < children.length) {
      segments.push({ type: 'text', content: children.slice(lastIndex) });
    }

    return segments;
  }, [children]);

  // If no links found, just render plain text
  if (parts.length === 1 && parts[0].type === 'text') {
    return <Text {...textProps}>{children}</Text>;
  }

  return (
    <Text {...textProps}>
      {parts.map((part, index) => (
        <Fragment key={index}>
          {part.type === 'link' ? (
            <Anchor href={part.content} target="_blank" rel="noopener noreferrer">
              {part.content}
            </Anchor>
          ) : (
            part.content
          )}
        </Fragment>
      ))}
    </Text>
  );
}
