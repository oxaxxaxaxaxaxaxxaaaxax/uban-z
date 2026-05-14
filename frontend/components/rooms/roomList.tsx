'use client';

import { Box, Skeleton, Typography } from '@mui/material';
import RoomCard from './room';
import type { components } from '@/types/booking';
import styles from './roomList.module.scss';

type Room = components['schemas']['Room'];

interface RoomListProps {
    rooms: Room[];
    isLoading?: boolean;
    emptyMessage?: string;
    mode?: 'full' | 'compact';         
    onRoomClick?: (room: Room) => void;
    interactive?: boolean;
}

export default function RoomList({ rooms, isLoading = false,
    emptyMessage = 'Аудитории не найдены. Попробуйте изменить фильтры.',
    mode = 'full', onRoomClick, interactive = true
}: RoomListProps) {

    if (isLoading) {
        return (
            <Box className={styles.grid}>
                {[1, 2, 3, 4, 5, 6].map((i) => (
                    <Skeleton
                        key={i}
                        variant="rectangular"
                        height={120}
                        className={styles.skeleton}
                    />
                ))}
            </Box>
        );
    }

    if (rooms.length === 0) {
        return (
            <Box className={styles.empty}>
                <Typography variant="body1" color="text.secondary">
                    {emptyMessage}
                </Typography>
            </Box>
        );
    }

    return (
        <Box className={styles.grid}>
            {rooms.map((room) => (
                <RoomCard
                    key={room.id}
                    room={room}
                    mode={mode}
                    onClick={() => onRoomClick?.(room)}
                    interactive={interactive}
                />
            ))}
        </Box>
    );
}
