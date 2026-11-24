import { Critter } from './Critter';

export interface UserAvatarProps {
  /** The user's name: the seed when no id is given. */
  name: string;
  /** The user's id: a stable seed, so the critter survives a rename. */
  seed?: string;
  /** Avatar size - defaults to 'md' */
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  /** A ring in the critter's colour (e.g. the header's own avatar). */
  ring?: boolean;
}

const PX = { xs: 16, sm: 26, md: 38, lg: 56, xl: 84 } as const;

/**
 * UserAvatar - who someone is, as their lab critter
 *
 * Draws the user's critter (see Critter): the same user always gets the
 * same one. Seeded by the id when given, else by the name.
 *
 * @example
 * <UserAvatar name={user.name} seed={user.id} size="md" />
 */
export function UserAvatar({ name, seed, size = 'md', ring = false }: UserAvatarProps) {
  return <Critter seed={seed || name} size={PX[size]} ring={ring} />;
}
