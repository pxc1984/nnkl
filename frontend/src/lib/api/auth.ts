import axios from "axios";

import { api } from "$lib/api/client";
import type {
  AuthSession,
  LoginPayload,
  RegisterPayload,
  TokenPair,
  UserProfile,
} from "$lib/auth/types";

type ApiErrorPayload = {
  message?: string;
};

export async function registerUser(
  payload: RegisterPayload,
): Promise<UserProfile> {
  const response = await api.post<UserProfile>(
    "/api/v1/auth/register",
    payload,
  );
  return response.data;
}

export async function loginUser(payload: LoginPayload): Promise<AuthSession> {
  const response = await api.post<AuthSession>("/api/v1/auth/login", payload);
  return response.data;
}

export async function refreshAuthTokens(
  refreshToken: string,
): Promise<TokenPair> {
  const response = await api.post<TokenPair>("/api/v1/auth/refresh", {
    refreshToken,
  });
  return response.data;
}

export async function getCurrentUser(): Promise<UserProfile> {
  const response = await api.get<UserProfile>("/api/v1/auth/me");
  return response.data;
}

export async function logoutCurrentSession(
  refreshToken?: string,
): Promise<void> {
  await api.post("/api/v1/auth/logout", refreshToken ? { refreshToken } : {});
}

export async function logoutAllSessions(): Promise<void> {
  await api.post("/api/v1/auth/logout-all");
}

export function getApiErrorMessage(
  error: unknown,
  fallbackMessage: string,
): string {
  if (axios.isAxiosError<ApiErrorPayload>(error)) {
    return error.response?.data?.message ?? fallbackMessage;
  }

  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallbackMessage;
}

export function isUnauthorizedError(error: unknown): boolean {
  return axios.isAxiosError(error) && error.response?.status === 401;
}
