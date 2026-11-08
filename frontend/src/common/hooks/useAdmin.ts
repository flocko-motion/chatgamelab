import { useAuth } from "@/providers/AuthProvider";
import { isAdmin } from "@/common/lib/roles";

export function useAdmin() {
  const { backendUser } = useAuth();
  return { isAdmin: isAdmin(backendUser) };
}
