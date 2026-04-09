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

// real: contact to api gateway
export async function mockLogin(email: string, password: string): Promise<{ success: boolean; user?: User }> { 
  await new Promise(res => setTimeout(res, 800))

  // демо проверка
  if (email === MOCK_USER.username && password === '123456') {
    return { success: true, user: MOCK_USER }
  }

  return { success: false }    
}   
