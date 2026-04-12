export interface User {
    id: string,
    username: string,
    fullName: string,
    role: string,
}

const MOCK_USER: User = {
    id: '1',
    username: 'i.ivanov',
    fullName: 'Иванов Иван Иванович',
    role: 'студент',
}

const _MOCK_TOKEN = 'mock_token_123'

// real: contact to api gateway
export async function mockLogin(email: string, password: string): Promise<{ success: boolean; user?: User }> { 
  await new Promise(res => setTimeout(res, 800));

  if (email === MOCK_USER.username && password === '123456') {
    if (process.env.NODE_ENV === 'development') {
        document.cookie = 'session_token=mock_token_123; path=/; max-age=3600';
    }
    return { success: true, user: MOCK_USER }
  }

  return { success: false }    
}


export function mockLogout(): void {
  // replace: fetch(...) to API Gateway
  document.cookie = 'session_token=; path=/; max-age=0';
}


export async function mockGetMe(token?: string): Promise<User | null> {
  await new Promise(res => setTimeout(res, 300));

  // server
  if (typeof document === 'undefined') {
    return token === 'mock_token_123' ? MOCK_USER : null;
  }

  // client
  const hasValidToken = document.cookie.includes('session_token=mock_token_123');
  return hasValidToken ? MOCK_USER : null;
}
