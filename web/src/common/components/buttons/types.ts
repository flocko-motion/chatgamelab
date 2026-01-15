/**
 * Common props interface for icon button components
 */
export interface IconButtonProps {
  onClick?: () => void;
  'aria-label': string;
  disabled?: boolean;
  loading?: boolean;
  size?: 'sm' | 'md' | 'lg';
}
