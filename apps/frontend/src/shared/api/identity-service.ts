import { apiRequest } from "@/shared/api/http-client";
import type { UserRole, UserSession } from "@/shared/types/domain";

interface RegisterCompanyInput {
  name: string;
  inn: string;
  ogrn: string;
}

interface RegisterUserInput {
  companyId: string;
  name: string;
  role: UserRole;
  login: string;
  password: string;
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
}

interface LoginResponse {
  token: string;
  user: UserResponse;
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
      name: input.name,
      role: input.role,
      login: input.login,
      password: input.password,
    },
  });
}

export async function login(input: LoginInput): Promise<LoginResponse> {
  return apiRequest<LoginResponse>("identity", "/auth/login", {
    method: "POST",
    body: input,
  });
}

export async function getCurrentUser(session: UserSession): Promise<UserResponse> {
  return apiRequest<UserResponse>("identity", "/users/me", { session });
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
    mode,
    updatedAt: new Date().toISOString(),
  };
}
