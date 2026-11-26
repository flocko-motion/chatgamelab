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
  /** If true, shows the simplified UI of a Guest in a workshop */
  isGuest?: boolean;
  /** If true, shows the minimal signed-out header (Contact + Language only, no user menu) */
  isSignedOut?: boolean;
  /** If true, staff/head has entered workshop mode (keep user bubble, show exit button) */
  isInWorkshopMode?: boolean;
  /** Name of the workshop when in workshop mode */
  workshopName?: string | null;
  /** Callback to exit workshop mode */
  onExitWorkshopMode?: () => void;
  /**
   * If true, never shows the phone's tab bar: a game fills the screen and
   * keeps no room for it, so the bar would cover its controls. The header's
   * burger keeps the full menu.
   */
  withoutTabBar?: boolean;
}
