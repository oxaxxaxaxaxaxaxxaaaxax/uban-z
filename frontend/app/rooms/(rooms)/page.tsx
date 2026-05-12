import Header from '@/components/header';
import BackButton from '@/components/backButton';
import RoomsContent from '@/components/rooms/roomsContent';

import styles from '../page.module.scss'

import { getMe } from '@/lib/api/auth';
import { getRooms } from '@/lib/api/booking';
import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';

export default async function RoomsPage() {
    const cookieStore = await cookies();
    const token = cookieStore.get(`session_token`)?.value;

    if (!token) {
        redirect(`/login`);
    }

    const user = await getMe(token);
    if (!user) {
        redirect(`/login`);
    }

    const roomsResult = await getRooms();
    const rooms = roomsResult.success && roomsResult.rooms ? roomsResult.rooms : [];

    return (
        <main className={styles.container}>
            <div className={styles.content}>
                <Header fullname={user.fullname} />
                <section className={styles.section}>
                    <BackButton fallback="/" />
                    <RoomsContent 
                        initialRooms={rooms}
                        interactive={false}
                    />
                </section>
            </div>
        </main>
    );
}
