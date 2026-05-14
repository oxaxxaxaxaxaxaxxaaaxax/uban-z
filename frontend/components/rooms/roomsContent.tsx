'use client';

import { useState, useMemo } from 'react';
import { Box } from '@mui/material';
import { useFilteredRooms, type RoomFilters } from '@/hooks/useFilteredRooms';
import RoomFiltersUI from '@/components/rooms/roomFilters';
import RoomList from '@/components/rooms/roomList';
import type { components } from '@/types/booking';
import styles from './roomsContent.module.scss';

type Room = components['schemas']['Room'];

interface RoomsContentProps {
    initialRooms: Room[];
    interactive?: boolean;
}

export default function RoomsContent({
    initialRooms,
    interactive = true
}: RoomsContentProps) {
    const [filters, setFilters] = useState<RoomFilters>({
        search: '',
        building: undefined,
        minCapacity: undefined,
    });

    const { filteredRooms } = useFilteredRooms(initialRooms, filters);

    const availableBuildings = useMemo(() => {
        const buildings = initialRooms
            .map(room => room.building)
            .filter((b): b is string => Boolean(b));
        return [...new Set(buildings)];
    }, [initialRooms]);

    return (
        <Box className={styles.container}>
            <RoomFiltersUI
                value={filters}
                onChange={setFilters}
                availableBuildings={availableBuildings}
            />
            <RoomList
                rooms={filteredRooms}
                mode="full"
                interactive={interactive}
            />
        </Box>
    );
}
