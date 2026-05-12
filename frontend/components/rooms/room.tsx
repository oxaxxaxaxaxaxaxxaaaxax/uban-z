'use client';

import { Card, CardContent, Typography, Box } from '@mui/material';
import LocationCityIcon from '@mui/icons-material/LocationCity';
import PeopleIcon from '@mui/icons-material/People';
import type { components } from '@/types/booking';
import styles from './room.module.scss';

type Room = components['schemas']['Room'];

interface RoomCardProps {
    room: Room;
    mode?: 'full' | 'compact'; // 'full' — для /rooms, 'compact' — для остального
    onClick?: () => void;      
    interactive?: boolean;
}

export default function RoomCard({ room, mode = 'full', onClick, interactive = true }: RoomCardProps) {
    const isCompact = mode === 'compact';
    const isClickable = typeof onClick === 'function';

    return (
        <Box
            className={`${styles.wrapper} ${isClickable ? styles.clickable : ''} ${!interactive ? styles.noHover : ''}`}
            onClick={onClick}
        >
            <Card className={styles.card}>
                <CardContent className={styles.content}>

                    <Box className={styles.header}>
                        <Typography variant="h6" className={styles.name}>
                            {room.name || `Аудитория №${room.id}`}
                        </Typography>
                    </Box>

                    {!isCompact && (
                        <Box className={styles.info}>
                            <Box className={styles.infoRow}>
                                <LocationCityIcon className={styles.icon} />
                                <Typography variant="body2" className={styles.text}>
                                    {room.building || 'Корпус не указан'}
                                </Typography>
                            </Box>

                            <Box className={styles.infoRow}>
                                <PeopleIcon className={styles.icon} />
                                <Typography variant="body2" className={styles.text}>
                                    Вместимость: {room.capacity ?? 0}
                                </Typography>
                            </Box>
                        </Box>
                    )}

                </CardContent>
            </Card>
        </Box>
    );
}
