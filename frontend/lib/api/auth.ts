import { apiRequest } from "./client";
import type { components } from '@/types/auth';
import { logger } from '../logger';

type LoginRequest = components['schemas']['LoginRequest'];
type LoginResponse = components['schemas']['LoginResponse'];
type RegisterRequest = components['schemas']['RegisterRequest'];
type UserResponse = components['schemas']['UserResponse'];

export async function register(login: string, password: string, role: string, fullName: string): Promise<{
    success: boolean,     error?: { status: number; message: string } 
}> {
    const { data, error } = await apiRequest<UserResponse>('/auth/register', {
        method: 'POST',
        body: JSON.stringify({ login, password, role, full_name: fullName } as RegisterRequest),
    });

    if (error) {
        logger.error('Register error', error);
        return { success: false, error };
    }

    if (!data?.login) {
        logger.error('Register error', error);
        return { success: false, error: { status: 500, message: 'Ошибка сервера' } };
    }
    logger.info('Register successful');
    return { success: true };
}

export async function login(username: string, password: string): Promise<{
                success: boolean, token?: string; error?: { status: number; message: string }
}> {  
    const { data, error } = await apiRequest<LoginResponse>('/auth/login', {
        method: 'POST',
        body: JSON.stringify({ login: username, password } as LoginRequest),
    });
    
    if (error) {
        logger.error('Authorization error', error);
        return { success: false, error };
    }

    if (!data?.token) {
        logger.error('Authorization error', error);
        return { success: false };
    }
    logger.info('Authorization successful, token received');
    return { success: true, token: data.token };
}


export async function logout(): Promise<{ success: boolean }> {
    document.cookie = 'session_token=; path=/; max-age=0';
    logger.info('Logout user successful')
    return { success: true };
  
    // const { error } = await apiRequest('/auth/logout', {
    //   method: 'POST',
    // });
  
    // if (error) {
    //   console.error('Logout error:', error);
    //   return { success: false };
    // }
  
    // return { success: true };
}


export interface User {
    fullname: string,
    role: string,
}

export async function getMe(token?: string): Promise<User | null> {
  const { data, error } = await apiRequest<UserResponse>('/auth/me', {
    method: 'GET',
    authToken: token,
  })

  if (error || !data) {
    logger.error('GetMe error', error)
    return null;
  }
    
  logger.info('GetMe successful')
  return {
    fullname: data.full_name ?? data.login ?? '', 
    role: data.role ?? '',         
  };
}
