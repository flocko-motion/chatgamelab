/**
 * Semantic Typography Components
 * 
 * These components encode information hierarchy - choose based on WHAT the text represents,
 * not how you want it to look.
 * 
 * Headings:
 * - PageTitle: Main page heading (h1)
 * - SectionTitle: Section heading within a page (h2)
 * - CardTitle: Card/panel heading (h3)
 * - ItemTitle: The name of one item in a list or grid (h3, small)
 * 
 * Text:
 * - BodyText: Primary readable content
 * - Label: Form field labels, metadata labels
 * - HelperText: Secondary/muted descriptions
 * - ErrorText: Error messages
 * - Kicker: A short label above or beside a heading
 */

export { PageTitle, type PageTitleProps } from './PageTitle';
export { SectionTitle, type SectionTitleProps } from './SectionTitle';
export { CardTitle, type CardTitleProps } from './CardTitle';
export { ItemTitle, type ItemTitleProps } from './ItemTitle';
export { BodyText, type BodyTextProps } from './BodyText';
export { Label, type LabelProps } from './Label';
export { HelperText, type HelperTextProps } from './HelperText';
export { ErrorText, type ErrorTextProps } from './ErrorText';
export { Kicker, type KickerProps } from './Kicker';
