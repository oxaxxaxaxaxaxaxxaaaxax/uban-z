'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import {
    Box, TextField, Button, Typography, Alert, CircularProgress,
    Autocomplete, MenuItem
} from '@mui/material';
import { TIME_SLOTS } from '@/lib/time-slots';
import { createBooking } from '@/lib/api/booking';
import type { components } from '@/types/booking';
import styles from './bookingForm.module.scss';

type Room = components['schemas']['Room'];

interface BookingFormProps {
    initialRooms: Room[];
    onSuccess?: () => void;
}

export default function BookingForm({ initialRooms, onSuccess }: BookingFormProps) {
    const router = useRouter();

    const [selectedRoom, setSelectedRoom] = useState<Room | null>(null);
    const [date, setDate] = useState('');
    const [startTime, setStartTime] = useState('');
    const [endTime, setEndTime] = useState('');

    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState<string | null>(null);
    const [conflictInfo, setConflictInfo] = useState<any>(null);

    const today = new Date().toISOString().split('T')[0];

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
            const startISO = `${date}T${startTime}:00`;
            const endISO = `${date}T${endTime}:00`;

            const res = await createBooking(selectedRoom.id, startISO, endISO);

            if (res.success && res.booking) {
                setSuccess('Бронирование успешно создано!');
                onSuccess?.();
                setSelectedRoom(null); setDate(''); setStartTime(''); setEndTime('');
            } else {
                setError(res.error?.message || 'Произошла ошибка');
                if (res.error?.status === 409) setConflictInfo(res.error.conflictInfo);
            }
        } catch {
            setError('Ошибка сети. Попробуйте позже.');
        } finally {
            setLoading(false);
        }
    };

    const availableTimeSlots = TIME_SLOTS.filter(slot => {
        if (!date) return TIME_SLOTS;
        const selectedDate = new Date(date);
        const todayDate = new Date(today);
        const isToday = selectedDate.toDateString() === todayDate.toDateString();
        if (!isToday) return true;

        const [slotHour, slotMinute] = slot.start.split(':').map(Number);
        const slotTime = new Date();
        slotTime.setHours(slotHour, slotMinute, 0, 0);
        return slotTime > new Date(Date.now() + 15 * 60 * 1000);
    });

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
                disabled={!date || availableTimeSlots.length === 0}
                helperText={date && availableTimeSlots.length === 0 ? 'Все занятия сегодня уже прошли' : ''}
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
                            {conflictInfo.type} ({conflictInfo.teacher}) для групп {conflictInfo.groups?.join(', ')}
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
