'use client';

import { useState } from 'react';
import {
  Box, Typography, Card, CardContent, Button, Alert,
  Dialog, DialogTitle, DialogContent, DialogActions, CircularProgress
} from '@mui/material';
import { cancelBooking } from '@/lib/api/booking';
import type { components } from '@/types/booking';
import styles from './myBookings.module.scss';

type Booking = components['schemas']['Booking'] & {
  room_name?: string;
  building?: string;
};

interface MyBookingsListProps {
  initialBookings: Booking[];
}

const formatBookingTime = (start?: string, end?: string): string => {
  if (!start || !end) return '—';

  const startDate = new Date(start);
  const endDate = new Date(end);

  const dateStr = startDate.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' });
  const dayStr = startDate.toLocaleDateString('ru-RU', { weekday: 'long' });
  const startTime = startDate.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  const endTime = endDate.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });

  return `${dateStr}, ${dayStr}, ${startTime}–${endTime}`;
};

const formatDate = (iso?: string) => {
  if (!iso) return '—';
  return new Date(iso).toLocaleDateString('ru-RU', {
    weekday: 'long', day: 'numeric', month: 'long',
    hour: '2-digit', minute: '2-digit'
  });
};

export default function MyBookingsList({ initialBookings }: MyBookingsListProps) {
  const [bookings, setBookings] = useState<Booking[]>(initialBookings);
  const [cancelingId, setCancelingId] = useState<number | null>(null);
  const [confirmDialog, setConfirmDialog] = useState<{ open: boolean; booking: Booking | null }>({
    open: false,
    booking: null
  });
  const [notification, setNotification] = useState<{ type: 'success' | 'error'; message: string } | null>(null);

  const handleOpenConfirm = (booking: Booking) => {
    setConfirmDialog({ open: true, booking });
  };

  const handleCloseConfirm = () => {
    setConfirmDialog({ open: false, booking: null });
  };

  const handleCancel = async () => {
    const booking = confirmDialog.booking;
    if (!booking?.id) return;

    setCancelingId(booking.id);
    try {
      const res = await cancelBooking(booking.id);
      if (res.success) {
        setBookings(prev => prev.filter(b => b.id !== booking.id));
        setNotification({ type: 'success', message: 'Бронирование отменено' });
        handleCloseConfirm();
        setTimeout(() => setNotification(null), 3000);
      } else {
        setNotification({ type: 'error', message: res.error?.message || 'Не удалось отменить' });
      }
    } catch {
      setNotification({ type: 'error', message: 'Ошибка сети' });
    } finally {
      setCancelingId(null);
    }
  };

  if (bookings.length === 0) {
    return (
      <Box className={styles.empty}>
        <Typography>У вас пока нет активных бронирований</Typography>
        <Button variant="contained" href="/booking/create" className={styles.createBtn}>
          Создать бронь
        </Button>
      </Box>
    );
  }

  return (
    <>
      {/* Уведомления */}
      {notification && (
        <Alert
          severity={notification.type}
          className={styles.notification}
          onClose={() => setNotification(null)}
        >
          {notification.message}
        </Alert>
      )}

      {/* Список броней */}
      <Box className={styles.list}>
        {bookings.map((booking) => (
          <Card key={booking.id} className={styles.card}>
            <CardContent className={styles.cardContent}>
              <Box className={styles.cardHeader}>
                <Typography variant="h6" className={styles.roomName}>
                  Аудитория {booking.room_name || '${booking.room_id}'}
                </Typography>
                <Typography className={styles.building}>
                  {booking.building}
                </Typography>
              </Box>

              <Box className={styles.timeInfo}>
                <Typography className={styles.timeLabel}>
                  {formatBookingTime(booking.start_time, booking.end_time)}
                </Typography>
              </Box>

              <Box className={styles.actions}>
                <Button
                  variant="outlined"
                  color="error"
                  size="small"
                  onClick={() => handleOpenConfirm(booking)}
                  disabled={cancelingId === booking.id}
                  className={styles.cancelBtn}
                >
                  {cancelingId === booking.id ? (
                    <CircularProgress size={16} color="inherit" />
                  ) : 'Отменить'}
                </Button>
              </Box>
            </CardContent>
          </Card>
        ))}
      </Box>

      {/* Окно подтверждения */}
      <Dialog open={confirmDialog.open} onClose={handleCloseConfirm} className={styles.dialog}>
        <DialogTitle>Подтвердите отмену</DialogTitle>
        <DialogContent>
          <Typography>
            Вы действительно хотите отменить бронь: аудитория{' '}
            <strong>{confirmDialog.booking?.room_name || confirmDialog.booking?.room_id}</strong>,{' '}
            {formatDate(confirmDialog.booking?.start_time)}?
          </Typography>
          <Typography variant="body2" className={styles.dialogHint}>
            Это действие нельзя будет отменить.
          </Typography>
        </DialogContent>
        <DialogActions className={styles.dialogActions}>
          <Button onClick={handleCloseConfirm} disabled={!!cancelingId}>
            Назад
          </Button>
          <Button
            onClick={handleCancel}
            color="error"
            variant="contained"
            disabled={!!cancelingId}
          >
            {cancelingId ? 'Отменяем...' : 'Да, отменить'}
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}
