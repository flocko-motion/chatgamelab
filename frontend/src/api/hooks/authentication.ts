/**
 * Signing in and out, and who is calling. These are plain functions, not
 * hooks: the auth bootstrap and the forms call them imperatively, with their
 * own error handling, outside the query cache.
 *
 * The auth provider imports this file directly rather than `@/api/hooks`,
 * since some hooks there read `useAuth` and that would close an import cycle.
 */
import { api } from "../client/api";
import type {
  CheckNameResp,
  CompleteRegistrationReq,
  GetCurrentUserResp,
  UserView,
} from "../generated/api.gen";

/** Who is calling (GET /api/users/me): anonymous, registering or registered. */
export const fetchCaller = (): Promise<GetCurrentUserResp> =>
  api.getApiUsersMe().then((response) => response.data);

/** Completes registration of the signed-in account. */
export const registerAccount = (request: CompleteRegistrationReq): Promise<UserView> =>
  api.postApiAuthRegister(request).then((response) => response.data);

/** Whether a user name is free and not profane. */
export const checkName = (name: string): Promise<CheckNameResp> =>
  api.getApiAuthCheckName({ name }).then((response) => response.data);

/** Signs a Guest in with their token; the server sets the session cookie. */
export const signInGuest = (token: string): Promise<void> =>
  api.postApiAuthGuestLogin({ token }).then(() => undefined);

/** Exchanges an Auth0 ID token for a session. */
export const signInWithAuth0 = (idToken: string): Promise<void> =>
  api.postApiAuthAuth0Login({ id_token: idToken }).then(() => undefined);

/** Signs in as one of the preset dev accounts (dev server only). */
export const signInDev = (role: string): Promise<void> =>
  api.postApiAuthDevLogin({ role }).then(() => undefined);

/** Ends the session. */
export const signOut = (): Promise<void> =>
  api.postApiAuthLogout().then(() => undefined);
