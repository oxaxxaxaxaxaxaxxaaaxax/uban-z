'use client';

import { useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { useRouter } from 'next/navigation';
import { Alert, Box, Button, CircularProgress, Typography } from '@mui/material';
import { getScheduleImportStatus, type ScheduleImportStatus } from '@/lib/api/booking';

interface ScheduleImportGateProps {
    initialStatus?: ScheduleImportStatus;
    children: ReactNode;
}

export default function ScheduleImportGate({ initialStatus, children }: ScheduleImportGateProps) {
    const router = useRouter();
    const [status, setStatus] = useState<ScheduleImportStatus | undefined>(initialStatus);
    const isReady = status?.status === 'ready';
    const isFailed = status?.status === 'failed';

    useEffect(() => {
        if (isReady || isFailed) return;

        let cancelled = false;
        const interval = window.setInterval(async () => {
            const result = await getScheduleImportStatus();
            if (cancelled || !result.success || !result.importStatus) return;

            setStatus(result.importStatus);
            if (result.importStatus.status === 'ready') {
                router.refresh();
            }
        }, 3000);

        return () => {
            cancelled = true;
            window.clearInterval(interval);
        };
    }, [isFailed, isReady, router]);

    if (isReady) {
        return <>{children}</>;
    }

    if (isFailed) {
        return (
            <Alert
                severity="error"
                action={<Button color="inherit" size="small" onClick={() => router.refresh()}>Обновить</Button>}
            >
                Не удалось загрузить расписание: {status?.error || 'неизвестная ошибка'}
            </Alert>
        );
    }

    return (
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, py: 3 }}>
            <CircularProgress size={24} />
            <Typography>Расписание загружается, аудитории и бронирование скоро станут доступны.</Typography>
        </Box>
    );
}
