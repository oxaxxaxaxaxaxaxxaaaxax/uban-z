import Header from '@/components/header';
import BackButton from '@/components/backButton';
import BookingForm from '@/components/bookingForm';
import ScheduleImportGate from '@/components/scheduleImportGate';

import styles from '../page.module.scss'

import { getMe } from '@/lib/api/auth';
import { cookies } from 'next/headers';
import { getRooms, getScheduleImportStatus } from '@/lib/api/booking';
import { redirect } from 'next/navigation';

export default async function CreateBookingPage() {
    const cookieStore = await cookies();
    const token = cookieStore.get(`session_token`)?.value;

    if (!token) {
        redirect(`/login`);
    }

    const user = await getMe(token);
    if (!user) {
        redirect(`/login`);
    }

    const importStatusResult = await getScheduleImportStatus();
    const importStatus = importStatusResult.success ? importStatusResult.importStatus : undefined;
    const roomsResult = importStatus?.status === 'ready' ? await getRooms() : null;
    const initialRooms = roomsResult?.success && roomsResult.rooms ? roomsResult.rooms : [];

    return (
        <main className={styles.container}>
            <div className={styles.content}>
                <Header fullname={user.fullname} />
                <section>
                    <BackButton fallback="/" />
                    <ScheduleImportGate initialStatus={importStatus}>
                        <BookingForm
                            initialRooms={initialRooms}
                        />
                    </ScheduleImportGate>
                </section>
            </div>
        </main>
    );
}
