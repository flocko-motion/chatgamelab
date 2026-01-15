/**
 * User utility functions for avatar generation and name processing
 */

/**
 * Extracts initials from a user's name
 * Examples:
 * - "Janos" -> "J"
 * - "Janos Brodbeck" -> "JB"
 * - "Florian Blabla Metzger Nuger" -> "FB"
 * - "john doe" -> "JD" (handles lowercase)
 * 
 * @param name - User's full name
 * @returns 1-2 character initials string
 */
export function getUserInitials(name: string): string {
  if (!name || typeof name !== 'string') {
    return '?';
  }

  // Clean and split the name
  const parts = name.trim().split(/\s+/).filter(part => part.length > 0);
  
  if (parts.length === 0) {
    return '?';
  }

  if (parts.length === 1) {
    // Single name - use first character
    return parts[0].charAt(0).toUpperCase();
  }

  // Multiple names - use first character of first two parts
  return parts[0].charAt(0).toUpperCase() + parts[1].charAt(0).toUpperCase();
}

/**
 * Generates a consistent color based on user's name
 * Uses a hash function to ensure the same name always gets the same color
 * 
 * @param name - User's name to hash
 * @returns Mantine color name from a predefined palette
 */
export function getUserAvatarColor(name: string): string {
  if (!name || typeof name !== 'string') {
    return 'gray';
  }

  // Predefined color palette that works well with the theme
  // Only use colors that are guaranteed to exist in Mantine
  const colors = [
    'blue', 'cyan', 'teal', 'green', 'lime', 
    'yellow', 'orange', 'red', 'pink', 'purple',
    'indigo', 'violet', 'gray'
  ];

  // Simple hash function
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    const char = name.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash; // Convert to 32-bit integer
  }

  // Use absolute value and modulo to get index
  const index = Math.abs(hash) % colors.length;
  return colors[index];
}

/**
 * Determines if white text should be used for better contrast
 * based on the avatar background color
 * 
 * @param color - Mantine color name
 * @returns boolean indicating if white text should be used
 */
export function shouldUseWhiteText(color: string): boolean {
  // Dark colors where white text provides better contrast
  const darkColors = ['blue', 'cyan', 'teal', 'green', 'indigo', 'purple', 'violet'];
  return darkColors.includes(color);
}
