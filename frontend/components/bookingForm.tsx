'use client';

import { useEffect, useState } from 'react';
import {
    Box, TextField, Button, Typography, Alert, CircularProgress,
    Autocomplete, MenuItem
} from '@mui/material';
import { TIME_SLOTS } from '@/lib/time-slots';
import { createBooking, getRoomSchedule } from '@/lib/api/booking';
import type { components } from '@/types/booking';
import styles from './bookingForm.module.scss';

type Room = components['schemas']['Room'];
type ScheduleItem = components['schemas']['ScheduleItem'];

interface ConflictInfo {
    message?: string;
    type?: string;
    teacher?: string;
    groups?: string[];
}

interface BookingFormProps {
    initialRooms: Room[];
    onSuccess?: () => void;
}

const toDateInputValue = (value: Date) => {
    const offsetDate = new Date(value.getTime() - value.getTimezoneOffset() * 60_000);
    return offsetDate.toISOString().split('T')[0];
};

const toLocalISOString = (dateValue: string, timeValue: string) => {
    return new Date(`${dateValue}T${timeValue}:00`).toISOString();
};

const overlaps = (startA: number, endA: number, startB: number, endB: number) => {
    return startA < endB && startB < endA;
};

export default function BookingForm({ initialRooms, onSuccess }: BookingFormProps) {
    const [selectedRoom, setSelectedRoom] = useState<Room | null>(null);
    const [date, setDate] = useState('');
    const [startTime, setStartTime] = useState('');
    const [endTime, setEndTime] = useState('');
    const [schedule, setSchedule] = useState<ScheduleItem[]>([]);

    const [loading, setLoading] = useState(false);
    const [scheduleLoading, setScheduleLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [scheduleError, setScheduleError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);
    const [conflictInfo, setConflictInfo] = useState<ConflictInfo | null>(null);

    const today = toDateInputValue(new Date());

    useEffect(() => {
        let cancelled = false;

        setStartTime('');
        setEndTime('');
        setSchedule([]);
        setScheduleError(null);

        if (!selectedRoom?.id) {
            return;
        }

        setScheduleLoading(true);
        getRoomSchedule(selectedRoom.id)
            .then((res) => {
                if (cancelled) return;
                if (res.success) {
                    setSchedule(res.schedule || []);
                    return;
                }
                setScheduleError(res.error?.message || 'Не удалось загрузить расписание аудитории');
            })
            .catch(() => {
                if (!cancelled) setScheduleError('Не удалось загрузить расписание аудитории');
            })
            .finally(() => {
                if (!cancelled) setScheduleLoading(false);
            });

        return () => {
            cancelled = true;
        };
    }, [selectedRoom?.id]);

    const handleStartTimeChange = (val: string) => {
        setStartTime(val);
        const slot = TIME_SLOTS.find(s => s.start === val);
        if (slot) setEndTime(slot.end);
    };

    const handleSubmit = async (e: React.SyntheticEvent<HTMLFormElement>) => {
        e.preventDefault();
        setError(null); setSuccess(null); setConflictInfo(null);

        if (!selectedRoom?.id) { setError('Аудитория не выбрана'); return; }

        setLoading(true);
        try {
            const startISO = toLocalISOString(date, startTime);
            const endISO = toLocalISOString(date, endTime);

            const res = await createBooking(selectedRoom.id, startISO, endISO);

            if (res.success && res.booking) {
                setSuccess('Бронирование успешно создано!');
                onSuccess?.();
                setSelectedRoom(null); setDate(''); setStartTime(''); setEndTime('');
            } else {
                setError(res.error?.message || 'Произошла ошибка');
                if (res.error?.status === 409) setConflictInfo(res.error.conflictInfo ?? null);
            }
        } catch {
            setError('Ошибка сети. Попробуйте позже.');
        } finally {
            setLoading(false);
        }
    };

    const availableTimeSlots = TIME_SLOTS.filter(slot => {
        if (!date || !selectedRoom?.id || scheduleLoading || scheduleError) return false;
        const selectedDate = new Date(date);
        const todayDate = new Date(today);
        const isToday = selectedDate.toDateString() === todayDate.toDateString();
        if (isToday) {
            const [slotHour, slotMinute] = slot.start.split(':').map(Number);
            const slotTime = new Date();
            slotTime.setHours(slotHour, slotMinute, 0, 0);
            if (slotTime <= new Date(Date.now() + 15 * 60 * 1000)) {
                return false;
            }
        }

        const slotStart = new Date(toLocalISOString(date, slot.start)).getTime();
        const slotEnd = new Date(toLocalISOString(date, slot.end)).getTime();

        return !schedule.some((item) => {
            if (!item.start_time || !item.end_time) return false;
            const itemDate = toDateInputValue(new Date(item.start_time));
            if (itemDate !== date) return false;
            return overlaps(slotStart, slotEnd, new Date(item.start_time).getTime(), new Date(item.end_time).getTime());
        });
    });

    const timeHelperText = (() => {
        if (!selectedRoom) return 'Сначала выберите аудиторию';
        if (!date) return 'Сначала выберите дату';
        if (scheduleLoading) return 'Загружаем расписание аудитории...';
        if (scheduleError) return scheduleError;
        if (availableTimeSlots.length === 0) return 'На выбранную дату свободных интервалов нет';
        return '';
    })();

    return (
        <Box component="form" onSubmit={handleSubmit} className={styles.form}>
            <Typography variant="h5" className={styles.title}>Создать бронь</Typography>

            <Autocomplete
                options={initialRooms}
                getOptionLabel={(room) => `${room.name || `Аудитория ${room.id}`}`}
                value={selectedRoom}
                onChange={(_, newValue) => setSelectedRoom(newValue)}
                renderInput={(params) => (
                    <TextField {...params} label="Аудитория" required className={styles.field} />
                )}
                isOptionEqualToValue={(option, value) => option.id === value?.id}
            />

            <TextField
                label="Дата" type="date" value={date}
                onChange={(e) => { setDate(e.target.value); setStartTime(''); setEndTime(''); }}
                fullWidth required className={styles.field}
                slotProps={{ inputLabel: { shrink: true }, htmlInput: { min: today } }}
            />

            <TextField
                select label="Время" value={startTime}
                onChange={(e) => handleStartTimeChange(e.target.value)}
                fullWidth required className={styles.field}
                disabled={!selectedRoom || !date || scheduleLoading || Boolean(scheduleError) || availableTimeSlots.length === 0}
                helperText={timeHelperText}
            >
                {availableTimeSlots.map((slot) => (
                    <MenuItem key={slot.start} value={slot.start}>{slot.label}</MenuItem>
                ))}
            </TextField>

            {error && (
                <Alert severity="error" className={styles.alert} onClose={() => setError(null)}>
                    {error}
                    {conflictInfo && (
                        <Typography variant="body2" sx={{ mt: 1, fontStyle: 'italic' }}>
                            {conflictInfo.type || conflictInfo.message || 'Конфликт расписания'}
                            {conflictInfo.teacher && ` (${conflictInfo.teacher})`}
                            {conflictInfo.groups?.length ? ` для групп ${conflictInfo.groups.join(', ')}` : ''}
                        </Typography>
                    )}
                </Alert>
            )}

            {success && (
                <Alert severity="success" className={styles.alert} onClose={() => setSuccess(null)}>
                    {success}
                </Alert>
            )}

            <Button
                type="submit" variant="contained"
                disabled={loading || !selectedRoom || !date || !startTime || !endTime}
                className={styles.submitBtn}
                startIcon={loading && <CircularProgress size={20} color="inherit" />}
            >
                {loading ? 'Создаём...' : 'Забронировать'}
            </Button>
        </Box>
    );
}
