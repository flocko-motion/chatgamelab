/**
 * Role definitions and utilities for role-based access control.
 *
 * Roles are ordered by privilege level (lowest to highest):
 * - participant (0): Workshop participant, anonymous guest users
 * - staff (1): Institution staff, workshop facilitators
 * - head (2): Institution owner/head
 * - admin (3): Platform administrator (god-mode)
 *
 * Users without a role (null) are treated as having no privileges.
 */

import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import type { ObjUser, ObjUserRole } from "@/api/generated";

/**
 * Role values with numeric levels for easy comparison.
 * Higher value = more privileges.
 */
export const Role = {
  Participant: 0,
  Individual: 0.5,
  Staff: 1,
  Head: 2,
  Admin: 3,
} as const;

export type Role = (typeof Role)[keyof typeof Role];

/**
 * Role string values as they come from the API.
 */
export type RoleString =
  | "participant"
  | "individual"
  | "staff"
  | "head"
  | "admin";

/**
 * Maps API role strings to Role enum values.
 */
const ROLE_MAP: Record<RoleString, Role> = {
  participant: Role.Participant,
  individual: Role.Individual,
  staff: Role.Staff,
  head: Role.Head,
  admin: Role.Admin,
};

/**
 * Maps Role enum values to display-friendly labels.
 */
const ROLE_LABELS: Record<Role, string> = {
  [Role.Participant]: "Participant",
  [Role.Individual]: "Individual",
  [Role.Staff]: "Workshop Leader",
  [Role.Head]: "Head of Organization",
  [Role.Admin]: "Administrator",
};

/**
 * Parses a role string from the API into a Role enum value.
 * Returns undefined for unknown/missing roles.
 */
export function parseRole(
  roleString: string | undefined | null,
): Role | undefined {
  if (!roleString) return undefined;
  return ROLE_MAP[roleString as RoleString];
}

/**
 * Gets the Role enum value from a user object.
 * Returns undefined if user has no role.
 */
export function getUserRole(
  user: ObjUser | null | undefined,
): Role | undefined {
  if (!user?.role?.role) return undefined;
  return parseRole(user.role.role);
}

/**
 * Gets the role string from a user object.
 */
export function getUserRoleString(
  user: ObjUser | null | undefined,
): RoleString | undefined {
  return user?.role?.role as RoleString | undefined;
}

/**
 * Gets the UserRole object from a user.
 */
export function getUserRoleDetails(
  user: ObjUser | null | undefined,
): ObjUserRole | undefined {
  return user?.role;
}

/**
 * Gets the display label for a role.
 * Accepts Role enum, role string from API, or any string.
 */
export function getRoleLabel(role: Role | string | undefined | null): string {
  if (role === undefined || role === null || role === "") return "Guest";

  if (typeof role === "string") {
    const parsed = parseRole(role);
    return parsed !== undefined ? ROLE_LABELS[parsed] : "Guest";
  }

  return ROLE_LABELS[role] ?? "Guest";
}

/**
 * Checks if user has at least the minimum required role.
 *
 * @example
 * hasMinRole(user, Role.Staff) // true if staff, head, or admin
 * hasMinRole(user, Role.Admin) // true only if admin
 */
export function hasMinRole(
  user: ObjUser | null | undefined,
  minRole: Role,
): boolean {
  const userRole = getUserRole(user);
  if (userRole === undefined) return false;
  return userRole >= minRole;
}

/**
 * Checks if user has exactly the specified role.
 */
export function hasRole(user: ObjUser | null | undefined, role: Role): boolean {
  const userRole = getUserRole(user);
  return userRole === role;
}

/**
 * Checks if user has any of the specified roles.
 */
export function hasAnyRole(
  user: ObjUser | null | undefined,
  roles: Role[],
): boolean {
  const userRole = getUserRole(user);
  if (userRole === undefined) return false;
  return roles.includes(userRole);
}

/**
 * Checks if user is an admin.
 */
export function isAdmin(user: ObjUser | null | undefined): boolean {
  return hasRole(user, Role.Admin);
}

/**
 * Checks if user is at least a head (head or admin).
 */
export function isAtLeastHead(user: ObjUser | null | undefined): boolean {
  return hasMinRole(user, Role.Head);
}

/**
 * Checks if user is at least staff (staff, head, or admin).
 */
export function isAtLeastStaff(user: ObjUser | null | undefined): boolean {
  return hasMinRole(user, Role.Staff);
}

/**
 * Checks if user has any role (is not a guest).
 */
export function hasAnyRoleAssigned(user: ObjUser | null | undefined): boolean {
  return getUserRole(user) !== undefined;
}

/**
 * Checks if user is a guest (no role assigned).
 */
export function isGuest(user: ObjUser | null | undefined): boolean {
  return !hasAnyRoleAssigned(user);
}

/**
 * Gets the user's institution ID if they have one.
 */
export function getUserInstitutionId(
  user: ObjUser | null | undefined,
): string | undefined {
  return user?.role?.institution?.id;
}

/**
 * Checks if user belongs to a specific institution.
 */
export function isInInstitution(
  user: ObjUser | null | undefined,
  institutionId: string,
): boolean {
  return getUserInstitutionId(user) === institutionId;
}

/**
 * Compares two roles and returns:
 * - negative if role1 < role2
 * - 0 if role1 === role2
 * - positive if role1 > role2
 */
export function compareRoles(role1: Role, role2: Role): number {
  return role1 - role2;
}

/**
 * Returns the Mantine color associated with a role string.
 */
export function getRoleColor(role: string | undefined | null): string {
  switch (role) {
    case "admin":
      return "red";
    case "head":
      return "violet";
    case "staff":
      return "blue";
    case "participant":
    case "individual":
    default:
      return "gray";
  }
}

/**
 * Hook that returns a translateRole function using i18n.
 * Looks up `auth:profile.roles.<role>`, falls back to getRoleLabel.
 *
 * @param fallbackForEmpty - text to return when role is empty/undefined.
 *   Defaults to "-".
 */
export function useTranslateRole(fallbackForEmpty = "-") {
  const { t } = useTranslation("auth");

  const translateRole = useCallback(
    (role: string | undefined | null): string => {
      if (!role) return fallbackForEmpty;
      const key = `profile.roles.${role.toLowerCase()}`;
      const translated = t(key);
      // If the key is not found, react-i18next returns the key itself
      return translated === key ? getRoleLabel(role) : translated;
    },
    [t, fallbackForEmpty],
  );

  return translateRole;
}
