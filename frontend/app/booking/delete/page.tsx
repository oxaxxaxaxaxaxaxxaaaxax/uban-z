import Header from '@/components/header';
import BackButton from '@/components/backButton';
import MyBookingsList from '@/components/myBookings';

import styles from '../page.module.scss'

import { getMe } from '@/lib/api/auth';
import { cookies } from 'next/headers';
import { redirect } from 'next/navigation';
import { getUserBookings, getRooms } from '@/lib/api/booking';

export default async function DeleteBookingPage() {
    const cookieStore = await cookies();
    const token = cookieStore.get(`session_token`)?.value;

    if (!token) {
        redirect(`/login`);
    }

    const user = await getMe(token);
    if (!user) {
        redirect(`/login`);
    }

    const result = await getUserBookings();
    const bookings = result.bookings || [];

    const roomsResult = await getRooms();
    const initialRooms = roomsResult.success && roomsResult.rooms ? roomsResult.rooms : [];

    return (
        <main className={styles.container}>
            <div className={styles.content}>
                <Header fullname={user.fullname} />
                <section>
                    <BackButton fallback="/" />
                    <MyBookingsList initialBookings={bookings} />
                </section>
            </div>
        </main>
    );
}
