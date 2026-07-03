export type UserProfile = {
  id: string;
  email: string;
  name?: string;
  role: "admin" | "guest";
  emailVerified: boolean;
  avatarUrl?: string | null;
  lastLoginAt?: string | null;
  createdAt: string;
  updatedAt: string;
};

export type TokenPair = {
  accessToken: string;
  refreshToken: string;
  expiresAt: string;
};

export type AuthSession = TokenPair & {
  user: UserProfile;
};

export type LoginPayload = {
  email: string;
  password: string;
};

export type RegisterPayload = {
  email: string;
  name: string;
  password: string;
};
