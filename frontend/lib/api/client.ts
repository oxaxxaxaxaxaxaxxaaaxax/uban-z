// replace in real NEXT_PUBLIC_API_GATEWAY_URL in .env file
const API_BASE = process.env.NEXT_PUBLIC_API_GATEWAY_URL;

export async function apiRequest<T>(path: string, options: RequestInit = {}): Promise<{ data?: T; error?: { status: number; message: string } }> {
    try {
        const response = await fetch(`${API_BASE}${path}`, {
            headers: { 'Content-Type': 'application/json' },
            credentials: 'include',
            ...options,
        });

        if (!response.ok) {
            return {
                error: {
                    status: response.status,
                    message: response.statusText
                }
            };
        }

        const contentType = response.headers.get('content-type');
        if (contentType?.includes('application/json')) {
            const data = await response.json();
            return { data };
        }
        return { data: undefined as T };
    
    } catch (e) {
        return { 
            error: { 
                status: 0,
                message: 'Network error' 
            } 
        };
    }
}
