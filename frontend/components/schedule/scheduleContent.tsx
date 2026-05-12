'use client';

import { useState, useMemo } from 'react';
import { Box, Modal, Fade, CircularProgress } from '@mui/material';
import { useFilteredRooms, type RoomFilters } from '@/hooks/useFilteredRooms';
import RoomFiltersUI from '@/components/rooms/roomFilters';
import RoomList from '@/components/rooms/roomList';
import ScheduleTable from '@/components/schedule/scheduleView';
import { getRoomSchedule } from '@/lib/api/booking';
import type { components } from '@/types/booking';
import styles from './scheduleContent.module.scss';

type Room = components['schemas']['Room'];
type ScheduleItem = components['schemas']['ScheduleItem'];

interface ScheduleContentProps {
    initialRooms: Room[];
}

export default function ScheduleContent({ initialRooms }: ScheduleContentProps) {
    const [filters, setFilters] = useState<RoomFilters>({
        search: '', building: undefined, minCapacity: undefined
    });

    const [selectedRoom, setSelectedRoom] = useState<Room | null>(null);
    const [schedule, setSchedule] = useState<ScheduleItem[]>([]);
    const [loadingSchedule, setLoadingSchedule] = useState(false);

    const { filteredRooms } = useFilteredRooms(initialRooms, filters);
    const availableBuildings = useMemo(() =>
        [...new Set(initialRooms.map(r => r.building).filter(Boolean) as string[])],
        [initialRooms]);

    const handleRoomClick = async (room: Room) => {
        if (!room.id) return;
        setSelectedRoom(room);
        setLoadingSchedule(true);
        setSchedule([]);

        try {
            const res = await getRoomSchedule(room.id);
            if (res.success && res.schedule) setSchedule(res.schedule);
        } catch (err) {
            console.error('Failed to load schedule:', err);
        } finally {
            setLoadingSchedule(false);
        }
    };

    const handleCloseModal = () => {
        setSelectedRoom(null);
        setSchedule([]);
    };

    return (
        <>
            <RoomFiltersUI
                value={filters}
                onChange={setFilters}
                availableBuildings={availableBuildings}
            />

            <RoomList
                rooms={filteredRooms}
                mode="compact"
                onRoomClick={handleRoomClick}
            />

            <Modal
                open={!!selectedRoom}
                onClose={handleCloseModal}
                closeAfterTransition
                slotProps={{
                    backdrop: {
                        className: styles.backdrop,
                    },
                }}
            >
                <Fade in={!!selectedRoom}>
                    <Box className={styles.modalContainer}>
                        {loadingSchedule ? (
                            <Box className={styles.loaderContainer}>
                                <CircularProgress className={styles.loaderIcon} />
                            </Box>
                        ) : selectedRoom ? (
                            <ScheduleTable
                                schedule={schedule}
                                roomName={selectedRoom.name || `Аудитория ${selectedRoom.id}`}
                                onClose={handleCloseModal}
                            />
                        ) : null}
                    </Box>
                </Fade>
            </Modal>
        </>
    );
}
