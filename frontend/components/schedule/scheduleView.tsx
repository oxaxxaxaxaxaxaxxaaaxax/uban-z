'use client';

import { Box, Typography, IconButton } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import type { components } from '@/types/booking';
import { TIME_SLOTS } from '@/lib/time-slots'
import styles from './scheduleView.module.scss';

type ScheduleItem = components['schemas']['ScheduleItem'];

interface ScheduleViewProps {
    schedule: ScheduleItem[];
    roomName: string;
    onClose: () => void;
}

const DAYS_OF_WEEK = ['Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота'];

const TYPE_COLORS: Record<string, string> = {
    'лекция': 'lecture',
    'экзамен': 'lecture',
    'семинар': 'seminar',
    'практика': 'seminar',
    'лабораторная': 'lab',
};

const getTypeClass = (type?: string) => {
    if (!type) return '';
    const lowerType = type.toLowerCase();
    for (const [key, value] of Object.entries(TYPE_COLORS)) {
        if (lowerType.includes(key)) return value;
    }
    return '';
};

const getDayOfWeek = (iso?: string) => {
    if (!iso) return -1;
    const date = new Date(iso);
    const day = date.getDay();
    return day === 0 ? 6 : day - 1;
};


const getTimeSlotIndex = (time?: string) => {
    if (!time) return -1;

    const timePart = time.includes('T') ? time.split('T')[1] : time;
    const [hours, minutes] = timePart.split(':').map(Number);

    if (isNaN(hours) || isNaN(minutes)) {
        console.warn('getTimeSlotIndex: invalid time:', time);
        return -1;
    }

    const timeMinutes = hours * 60 + minutes;

    for (let i = 0; i < TIME_SLOTS.length; i++) {
        const [slotHours, slotMinutes] = TIME_SLOTS[i].start.split(':').map(Number);
        const slotTimeMinutes = slotHours * 60 + slotMinutes;

        if (Math.abs(timeMinutes - slotTimeMinutes) <= 15) {
            return i;
        }
    }

    return -1;
};

export default function ScheduleView({ schedule, roomName, onClose }: ScheduleViewProps) {
    const scheduleMatrix: (ScheduleItem | null)[][] = Array(DAYS_OF_WEEK.length)
        .fill(null)
        .map(() => Array(TIME_SLOTS.length).fill(null));

    schedule.forEach((item) => {
        const dayIndex = getDayOfWeek(item.start_time);
        const timeIndex = getTimeSlotIndex(item.start_time);
        if (dayIndex >= 0 && timeIndex >= 0) {
            scheduleMatrix[dayIndex][timeIndex] = item;
        }
    });

    if (schedule.length === 0) {
        return (
            <Box className={styles.container}>
                <Box className={styles.header}>
                    <Typography variant="h5" className={styles.title}>{roomName}</Typography>
                    <IconButton onClick={onClose} size="small"><CloseIcon /></IconButton>
                </Box>
                <Box className={styles.empty}>На эту неделю занятий не запланировано</Box>
            </Box>
        );
    }

    return (
        <Box className={styles.container}>
            <Box className={styles.header}>
                <Typography variant="h5" className={styles.title}>{roomName}</Typography>
                <IconButton onClick={onClose} size="small"><CloseIcon /></IconButton>
            </Box>

            <Box className={styles.tableWrapper}>
                <table className={styles.scheduleTable}>
                    <thead>
                        <tr>
                            <th className={styles.timeHeader}>Время</th>
                            {DAYS_OF_WEEK.map((day) => (
                                <th key={day} className={styles.dayHeader}>{day}</th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {TIME_SLOTS.map((slot, timeIndex) => (
                            <tr key={slot.label}>
                                <td className={styles.timeCell}>{slot.label}</td>
                                {DAYS_OF_WEEK.map((_, dayIndex) => {
                                    const item = scheduleMatrix[dayIndex][timeIndex];
                                    const typeClass = getTypeClass(item?.type);

                                    return (
                                        <td
                                            key={`${dayIndex}-${timeIndex}`}
                                            className={`${styles.cell} ${typeClass ? styles[typeClass] : ''}`}
                                        >
                                            {item ? (
                                                <Box className={styles.cellContent}>
                                                    <Typography className={styles.cellType}>{item.type}</Typography>

                                                    {item.teacher && (
                                                        <Typography className={styles.cellTeacher}>{item.teacher}</Typography>
                                                    )}

                                                    {item.groups_number && item.groups_number.length > 0 && (
                                                        <Typography className={styles.cellGroups}>
                                                            {item.groups_number.join(', ')}
                                                        </Typography>
                                                    )}
                                                </Box>
                                            ) : (
                                                <span className={styles.emptyCell}>—</span>
                                            )}
                                        </td>
                                    );
                                })}
                            </tr>
                        ))}
                    </tbody>
                </table>
            </Box>
        </Box>
    );
}
