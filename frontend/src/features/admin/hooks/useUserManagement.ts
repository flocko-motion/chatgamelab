import { useCallback, useState } from "react";
import {
  useDeleteUser,
  useRemoveUserRole,
  useSetUserRole,
  useUpdateUser,
  useUsers,
} from "@/api/hooks";
import type { UserView } from "@/api/types";
import { useAuth } from "@/providers/AuthProvider";
import { USER_LIST_PRESET } from "../lib/userListPresets";

/** The confirmation or form a user's action opens. */
type Pending =
  | { kind: "delete"; user: UserView }
  | { kind: "promote"; user: UserView }
  | { kind: "removeAdmin"; user: UserView }
  | { kind: "edit"; user: UserView };

/**
 * Everything the admins' user list does, for UserManagement to render: the
 * list (the server filters, sorts and pages), what may be done with each user,
 * and the dialogs for deleting, promoting, demoting and renaming.
 */
export function useUserManagement() {
  const { backendUser } = useAuth();
  const list = useUsers(USER_LIST_PRESET.list);

  const [pending, setPending] = useState<Pending | null>(null);
  const [editName, setEditName] = useState("");
  const close = useCallback(() => setPending(null), []);

  const deleteMutation = useDeleteUser();
  const makeAdminMutation = useSetUserRole();
  const removeAdminMutation = useRemoveUserRole();
  const updateMutation = useUpdateUser();
  const closeOnSuccess = { onSuccess: close };

  const userId = backendUser?.id;
  const isCurrentUser = useCallback(
    (user: UserView) => !!userId && user.id === userId,
    [userId],
  );
  /**
   * Someone else without an organisation may become an admin, another admin
   * lose the role: the user's role says which applies, the server's flag
   * whether the caller may.
   */
  const canPromote = useCallback(
    (user: UserView) =>
      !isCurrentUser(user) && user.mayChangeAnyRole && user.role?.role === "individual",
    [isCurrentUser],
  );
  const canRemoveAdmin = useCallback(
    (user: UserView) =>
      !isCurrentUser(user) && user.mayChangeAnyRole && user.role?.role === "admin",
    [isCurrentUser],
  );

  const userOf = (kind: Pending["kind"]) => (pending?.kind === kind ? pending.user : null);
  const deleting = userOf("delete");
  const promoting = userOf("promote");
  const demoting = userOf("removeAdmin");
  const editing = userOf("edit");

  return {
    list,
    isCurrentUser,
    canPromote,
    canRemoveAdmin,
    /** Renaming: the user's card right. */
    canEdit: (user: UserView) => user.mayEditCard,
    canDelete: (user: UserView) => user.mayDelete,

    requestDelete: (user: UserView) => setPending({ kind: "delete", user }),
    requestPromote: (user: UserView) => setPending({ kind: "promote", user }),
    requestRemoveAdmin: (user: UserView) => setPending({ kind: "removeAdmin", user }),
    requestEdit: (user: UserView) => {
      setEditName(user.name || "");
      setPending({ kind: "edit", user });
    },

    deleteModal: {
      user: deleting,
      opened: !!deleting,
      close,
      confirm: () => deleting && deleteMutation.mutate(deleting.id, closeOnSuccess),
      loading: deleteMutation.isPending,
    },
    promoteModal: {
      user: promoting,
      opened: !!promoting,
      close,
      confirm: () =>
        promoting && makeAdminMutation.mutate({ userId: promoting.id, role: "admin" }, closeOnSuccess),
      loading: makeAdminMutation.isPending,
    },
    removeAdminModal: {
      user: demoting,
      opened: !!demoting,
      close,
      confirm: () => demoting && removeAdminMutation.mutate(demoting.id, closeOnSuccess),
      loading: removeAdminMutation.isPending,
    },
    editModal: {
      user: editing,
      opened: !!editing,
      close,
      name: editName,
      setName: setEditName,
      unchanged: editName === (editing?.name || ""),
      confirm: () =>
        editing &&
        updateMutation.mutate({ id: editing.id, request: { name: editName } }, closeOnSuccess),
      loading: updateMutation.isPending,
    },
  };
}

export type UserManagementState = ReturnType<typeof useUserManagement>;
