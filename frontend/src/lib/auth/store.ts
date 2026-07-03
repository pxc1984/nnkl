import { writable } from "svelte/store";

import {
  getCurrentUser,
  loginUser,
  logoutCurrentSession,
  refreshAuthTokens,
  registerUser,
  isUnauthorizedError,
} from "$lib/api/auth";
import {
  clearStoredAuthSession,
  getStoredAuthSession,
  setStoredAuthSession,
} from "$lib/auth/storage";
import type {
  AuthSession,
  LoginPayload,
  RegisterPayload,
  UserProfile,
} from "$lib/auth/types";

type AuthState = {
  user: UserProfile | null;
  ready: boolean;
};

const initialSession = getStoredAuthSession();

export const authState = writable<AuthState>({
  user: initialSession?.user ?? null,
  ready: false,
});

export function hydrateAuthState(): void {
  authState.set({
    user: getStoredAuthSession()?.user ?? null,
    ready: true,
  });
}

export async function login(payload: LoginPayload): Promise<AuthSession> {
  const session = await loginUser(payload);
  persistSession(session);
  return session;
}

export async function signUp(payload: RegisterPayload): Promise<AuthSession> {
  await registerUser(payload);
  return login({ email: payload.email, password: payload.password });
}

export async function ensureAuthenticated(): Promise<boolean> {
  const storedSession = getStoredAuthSession();
  if (!storedSession?.user) {
    clearAuth();
    return false;
  }

  authState.set({ user: storedSession.user, ready: false });

  try {
    const user = await getCurrentUser();
    persistSession({ ...storedSession, user });
    return true;
  } catch (error) {
    if (isUnauthorizedError(error) && storedSession.refreshToken) {
      try {
        const tokenPair = await refreshAuthTokens(storedSession.refreshToken);
        const refreshedSession = { ...storedSession, ...tokenPair };
        setStoredAuthSession(refreshedSession);
        const user = await getCurrentUser();
        persistSession({ ...refreshedSession, user });
        return true;
      } catch {
        clearAuth();
        return false;
      }
    }

    if (isUnauthorizedError(error)) {
      clearAuth();
      return false;
    }

    authState.set({ user: storedSession.user, ready: true });
    return true;
  }
}

export async function logout(): Promise<void> {
  const storedSession = getStoredAuthSession();
  try {
    await logoutCurrentSession(storedSession?.refreshToken);
  } catch {
    // Local cleanup should still happen if the session is already gone server-side.
  }
  clearAuth();
}

function persistSession(session: AuthSession): void {
  setStoredAuthSession(session);
  authState.set({ user: session.user, ready: true });
}

function clearAuth(): void {
  clearStoredAuthSession();
  authState.set({ user: null, ready: true });
}
