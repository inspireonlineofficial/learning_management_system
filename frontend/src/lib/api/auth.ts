import { apiRequest, buildApiUrl } from "@/lib/api/client";
import { setStoredSession, type AuthUser, type Session } from "@/lib/session";

type AuthResponse = {
  access_token: string;
  refresh_token: string;
  user: AuthUser;
};

function normalizeUser(user: AuthUser): AuthUser {
  return {
    ...user,
    onboarded: user.onboarded ?? user.profile_complete,
  };
}

async function handleAuthResponse(res: AuthResponse): Promise<Session> {
  let user = res.user;
  if (!user) {
    const tempSession: Session = {
      accessToken: res.access_token,
      refreshToken: res.refresh_token,
      user: {} as any,
    };
    setStoredSession(tempSession);
    user = await getProfile();
  } else {
    user = normalizeUser(user);
  }
  return {
    accessToken: res.access_token,
    refreshToken: res.refresh_token,
    user,
  };
}

export async function loginUser(payload: { email: string; password: string }) {
  const res = await apiRequest<AuthResponse>("/v1/auth/login", {
    method: "POST",
    body: payload,
  });
  return handleAuthResponse(res);
}

export function registerUser(payload: { full_name: string; email: string; password: string }) {
  return apiRequest<{ message?: string; expires_in?: number }>("/v1/auth/register", {
    method: "POST",
    body: payload,
  });
}

export async function verifyOtp(payload: { email: string; otp: string }) {
  const res = await apiRequest<AuthResponse>("/v1/auth/verify-otp", {
    method: "POST",
    body: payload,
  });
  return handleAuthResponse(res);
}

export function resendOtp(payload: { email: string }) {
  return apiRequest<{ message?: string }>("/v1/auth/resend-otp", {
    method: "POST",
    body: payload,
  });
}

export function forgotPassword(payload: { email: string }) {
  return apiRequest<{ message?: string }>("/v1/auth/forgot-password", {
    method: "POST",
    body: payload,
  });
}

export function resetPassword(payload: { token: string; password: string }) {
  return apiRequest<{ message?: string }>("/v1/auth/reset-password", {
    method: "POST",
    body: payload,
  });
}

export function logoutUser(refreshToken: string) {
  return apiRequest<{ message?: string }>("/v1/auth/logout", {
    method: "POST",
    body: { refresh_token: refreshToken },
  }).catch(() => null);
}

export function getProfile() {
  return apiRequest<AuthUser>("/v1/auth/me", { auth: true }).then(normalizeUser);
}

export function updateProfile(payload: Record<string, unknown>) {
  return apiRequest<AuthUser>("/v1/auth/me", {
    method: "PATCH",
    auth: true,
    body: payload,
  }).then(normalizeUser);
}

export function getOAuthRedirectUrl(provider: string, returnTo?: string) {
  return buildApiUrl(`/v1/auth/oauth/${provider}`, returnTo ? { return: returnTo } : undefined);
}
