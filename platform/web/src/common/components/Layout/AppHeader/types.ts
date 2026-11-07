export interface NavItem {
  label: string;
  icon: React.ReactNode;
  onClick?: () => void;
  href?: string;
  active?: boolean;
  children?: NavItem[];
}

export interface AppHeaderProps {
  navItems?: NavItem[];
  onSettingsClick?: () => void;
  onProfileClick?: () => void;
  onApiKeysClick?: () => void;
  onLogoutClick?: () => void;
  userName?: string;
  /** If true, shows simplified participant UI (anonymous participant) */
  isParticipant?: boolean;
  /** If true, shows minimal guest header (Contact + Language only, no user menu) */
  isGuest?: boolean;
  /** If true, staff/head has entered workshop mode (keep user bubble, show exit button) */
  isInWorkshopMode?: boolean;
  /** Name of the workshop when in workshop mode */
  workshopName?: string | null;
  /** Callback to exit workshop mode */
  onExitWorkshopMode?: () => void;
}
