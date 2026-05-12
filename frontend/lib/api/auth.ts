import { apiRequest } from "./client";
import type { components } from '@/types/auth';

type LoginRequest = components['schemas']['LoginRequest'];
type LoginResponse = components['schemas']['LoginResponse'];


export async function login(username: string, password: string): Promise<{
                success: boolean, token?: string; error?: { status: number; message: string }
}> {  
    if (username === 'i.ivanov' && password === '123456') {
        const response = await fetch('/testData/login-success.json');
        const mockData = await response.json();
        return { 
            success: true, 
            token: mockData.token 
        };
    }
    
    return { 
        success: false, 
        error: { status: 401, message: 'Неверный логин или пароль' } 
    };

    // const { data, error } = await apiRequest<LoginResponse>('/auth/login', {
    //     method: 'POST',
    //     body: JSON.stringify({ login: username, password }),
    // });
    
    // if (error) {
    //     return { success: false, error };
    // }

    // if (!data?.token) {
    //     return { success: false };
    // }
    
    // return { success: true, token: data.token };
}


export async function logout(): Promise<{ success: boolean }> {
    document.cookie = 'session_token=; path=/; max-age=0';
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
    if (token === 'mock_token_123') {
        const isServer = typeof window === 'undefined';
        const url = isServer 
          ? `http://localhost:3000/testData/getme-user.json`
          : `/testData/getme-user.json`;                      

        const response = await fetch(url, { cache: 'no-store' });
        if (!response.ok) return null;
      
        const mockData = await response.json();
        return { 
            fullname: mockData.fullname,
            role: mockData.role,
        };
    } 
    return null;

  // const { data, error } = await apiRequest('/auth/me', {
  //   method: 'GET',
  // })

  // if (error || !data) {
  //   return null;
  // }

  // return {
  //   fullname: data.fullname, 
  //   role: data.role,         
  // };
}
