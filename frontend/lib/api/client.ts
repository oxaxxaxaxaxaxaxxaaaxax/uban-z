const API_BASE =
    typeof window === 'undefined'
        ? process.env.API_GATEWAY_URL ?? process.env.NEXT_PUBLIC_API_GATEWAY_URL ?? 'http://localhost:8080'
        : process.env.NEXT_PUBLIC_API_GATEWAY_URL ?? 'http://localhost:8080';

interface ApiRequestOptions extends RequestInit {
    authToken?: string;
}

function getSessionToken(): string | undefined {
    if (typeof document === 'undefined') return undefined;

    return document.cookie
        .split('; ')
        .find((row) => row.startsWith('session_token='))
        ?.split('=')[1];
}

function buildHeaders(options: ApiRequestOptions): Headers {
    const headers = new Headers(options.headers);

    if (!headers.has('Content-Type') && options.body) {
        headers.set('Content-Type', 'application/json');
    }

    const token = options.authToken ?? getSessionToken();
    if (token && !headers.has('Authorization')) {
        headers.set('Authorization', `Bearer ${decodeURIComponent(token)}`);
    }

    return headers;
}

function withoutAuthToken(options: ApiRequestOptions): RequestInit {
    const { authToken, ...fetchOptions } = options;
    void authToken;
    return fetchOptions;
}

async function errorMessage(response: Response): Promise<string> {
    const contentType = response.headers.get('content-type');
    if (contentType?.includes('application/json')) {
        const body = await response.json().catch(() => null);
        if (body && typeof body.error === 'string') return body.error;
        if (body && typeof body.message === 'string') return body.message;
    }

    return response.statusText;
}

export async function apiRequest<T>(path: string, options: ApiRequestOptions = {}): Promise<{ data?: T; error?: { status: number; message: string } }> {
    try {
        const fetchOptions = withoutAuthToken(options);
        const response = await fetch(`${API_BASE}${path}`, {
            headers: buildHeaders(options),
            credentials: 'include',
            cache: 'no-store',
            ...fetchOptions,
        });

        if (!response.ok) {
            return {
                error: {
                    status: response.status,
                    message: await errorMessage(response),
                }
            };
        }

        const contentType = response.headers.get('content-type');
        if (contentType?.includes('application/json')) {
            const data = await response.json();
            return { data };
        }
        return { data: undefined as T };
    
    } catch {
        return { 
            error: { 
                status: 0,
                message: 'Network error' 
            } 
        };
    }
}
