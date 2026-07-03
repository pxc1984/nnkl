import { browser } from "$app/environment";

import type { AuthSession } from "$lib/auth/types";

const AUTH_STORAGE_KEY = "auth.session";

export function getStoredAuthSession(): AuthSession | null {
  if (!browser) {
    return null;
  }

  const rawValue = window.localStorage.getItem(AUTH_STORAGE_KEY);
  if (!rawValue) {
    return null;
  }

  try {
    return JSON.parse(rawValue) as AuthSession;
  } catch {
    window.localStorage.removeItem(AUTH_STORAGE_KEY);
    return null;
  }
}

export function setStoredAuthSession(session: AuthSession): void {
  if (!browser) {
    return;
  }

  window.localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(session));
}

export function clearStoredAuthSession(): void {
  if (!browser) {
    return;
  }

  window.localStorage.removeItem(AUTH_STORAGE_KEY);
}
