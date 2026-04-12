import Header from '@/components/header';
import { mockGetMe } from '@/lib/testData';
import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';

export default async function MainPage() {
    const cookieStore = await cookies();
    const token = cookieStore.get(`session_token`)?.value;

    if (!token) {
        redirect(`/login`);
    }

    const user = await mockGetMe(token);
    if (!user) {
        redirect(`/login`);
    }

    return (
        <>
            <Header fullName={user.fullName} />
            <main>
                <h1>Это главная страница. Доступна только после авторизации!</h1>
            </main> 
        </>
    );
}
