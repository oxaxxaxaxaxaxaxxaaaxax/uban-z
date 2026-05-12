import Header from '@/components/header';
import { DASHBOARD_CARDS } from '@/lib/cards-config';
import Card from '@/components/card';

import { getMe } from '@/lib/api/auth';
import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';

import styles from './page.module.scss'

export default async function MainPage() {
    const cookieStore = await cookies();
    const token = cookieStore.get(`session_token`)?.value;

    if (!token) {
        redirect(`/login`);
    }

    const user = await getMe(token);
    if (!user) {
        redirect(`/login`);
    }

    return (
        <main className={styles.container}>
            <div className={styles.content}>
                <Header fullname={user.fullname}/>

                <div className={styles.grid}>
                    {DASHBOARD_CARDS.map((card) => (
                        <Card
                            key={card.href}
                            title={card.title}
                            icon={card.icon}
                            href={card.href}
                        />
                    ))}
                </div>
            </div>
        </main>
    );
}
