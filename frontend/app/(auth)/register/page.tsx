import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';
import styles from '../page.module.scss';
import RegisterForm from '@/components/registerForm';



export default async function LoginPage() {
    const cookieStore = await cookies();
    const token = cookieStore.get('session_token')?.value;

    if (token) {
        redirect('/');
    }

    return (
        <main className={styles.container}>
            <RegisterForm/>
        </main>
    );
}
