import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

const publicRoutes = ['/login'];

const STATIC_EXTENSIONS = ['.png', '.jpg', '.jpeg', '.svg', '.gif', '.ico', '.json']

export function middleware(request: NextRequest) {
    const { pathname } = request.nextUrl;
    
    const hasStaticExtension = STATIC_EXTENSIONS.some(ext => 
      pathname.toLowerCase().endsWith(ext)
    );

    if (hasStaticExtension) {
      return NextResponse.next();
    }

    const sessionToken = request.cookies.get('session_token')?.value;
    const isPublicRoute = publicRoutes.includes(pathname);

    if (!sessionToken && !isPublicRoute) {
        return NextResponse.redirect(new URL('/login', request.url));
    }

    if (sessionToken && isPublicRoute) {
        return NextResponse.redirect(new URL('/', request.url));
    }

    return NextResponse.next();
}


export const config = {
  matcher: [
    '/((?!api|_next/static|_next/image).*)',
  ],  
}

