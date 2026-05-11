import { apiRequest } from "@/shared/api/http-client";
import type { UserRole, UserSession } from "@/shared/types/domain";

interface RegisterCompanyInput {
  name: string;
  inn: string;
  ogrn: string;
}

interface RegisterUserInput {
  companyId?: string;
  companyInn?: string;
  companyOgrn?: string;
  name: string;
  role: UserRole;
  login: string;
  password: string;
  acceptedTerms: boolean;
  termsVersion: string;
}

interface LoginInput {
  login: string;
  password: string;
}

interface CompanyResponse {
  id: string;
  name: string;
  inn: string;
  ogrn: string;
  status: string;
  created_at: string;
}

interface UserResponse {
  id: string;
  company_id: string;
  name: string;
  role: UserRole;
  login: string;
  email_verified?: boolean;
}

interface LoginResponse {
  token: string;
  user: UserResponse;
}

interface VerifyEmailResponse {
  status: string;
  already_verified?: boolean;
}

interface ResendVerificationResponse {
  status: string;
  already_verified?: boolean;
}

interface ListUsersResponse {
  users: UserResponse[];
}

export async function registerCompany(input: RegisterCompanyInput): Promise<CompanyResponse> {
  return apiRequest<CompanyResponse>("identity", "/companies", {
    method: "POST",
    body: input,
  });
}

export async function registerUser(input: RegisterUserInput): Promise<UserResponse> {
  return apiRequest<UserResponse>("identity", "/users", {
    method: "POST",
    body: {
      company_id: input.companyId,
      company_inn: input.companyInn,
      company_ogrn: input.companyOgrn,
      name: input.name,
      role: input.role,
      login: input.login,
      password: input.password,
      accepted_terms: input.acceptedTerms,
      terms_version: input.termsVersion,
    },
  });
}

export async function login(input: LoginInput): Promise<LoginResponse> {
  return apiRequest<LoginResponse>("identity", "/auth/login", {
    method: "POST",
    body: input,
  });
}

export async function verifyEmail(token: string): Promise<VerifyEmailResponse> {
  const params = new URLSearchParams({ token });
  return apiRequest<VerifyEmailResponse>("identity", `/auth/verify-email?${params.toString()}`, {
    method: "GET",
  });
}

export async function resendVerification(loginValue: string): Promise<ResendVerificationResponse> {
  return apiRequest<ResendVerificationResponse>("identity", "/auth/resend-verification", {
    method: "POST",
    body: { login: loginValue },
  });
}

export async function getCurrentUser(session: UserSession): Promise<UserResponse> {
  return apiRequest<UserResponse>("identity", "/users/me", { session });
}

export async function promoteUserToAdmin(userId: string, session: UserSession): Promise<UserResponse> {
  return apiRequest<UserResponse>("identity", `/users/${userId}/promote-admin`, {
    method: "POST",
    session,
  });
}

export async function listUsers(session: UserSession): Promise<UserResponse[]> {
  const response = await apiRequest<ListUsersResponse>("identity", "/users", {
    method: "GET",
    session,
  });
  return response.users;
}

export function toUserSession(
  token: string,
  user: UserResponse,
  mode: UserSession["mode"],
): UserSession {
  return {
    accessToken: token,
    companyId: user.company_id,
    userId: user.id,
    role: user.role,
    name: user.name,
    login: user.login,
    emailVerified: user.email_verified,
    mode,
    updatedAt: new Date().toISOString(),
  };
}
