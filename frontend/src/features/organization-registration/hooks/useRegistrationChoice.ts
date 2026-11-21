import { useEffect, useState } from "react";
import {
  clearRegistrationIntent,
  getRegistrationIntent,
  type RegistrationIntent,
} from "../lib/registrationIntent";

/**
 * What a new account registers as: the choice made on the registration page
 * before signing in, used once, or none yet (null) for the person to choose.
 */
export function useRegistrationChoice() {
  const [choice, setChoice] = useState<RegistrationIntent | null>(getRegistrationIntent);
  useEffect(() => {
    clearRegistrationIntent();
  }, []);
  return [choice, setChoice] as const;
}
