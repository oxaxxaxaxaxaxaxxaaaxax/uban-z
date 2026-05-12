'use client';

import { useMemo } from 'react';
import type { components } from '@/types/booking';

type Room = components['schemas']['Room'];

export interface RoomFilters {
    search: string;
    building?: string;
    minCapacity?: number;     
}

interface UseFilteredRoomsResult {
    filteredRooms: Room[];
    hasActiveFilters: boolean;
}

export function useFilteredRooms(rooms: Room[], filters: RoomFilters): UseFilteredRoomsResult {
    const filteredRooms = useMemo(() => {
        return rooms.filter((room) => {
            if (filters.search) {
                const searchLower = filters.search.toLowerCase();
                const roomName = (room.name || '').toLowerCase();
                const roomId = String(room.id || '');

                if (!roomName.includes(searchLower) && !roomId.includes(searchLower)) {
                    return false;
                }
            }

            if (filters.building && filters.building !== 'all') {
                if (room.building !== filters.building) {
                    return false;
                }
            }

            if (filters.minCapacity) {
                if ((room.capacity || 0) < filters.minCapacity) {
                    return false;
                }
            }

            return true;
        });
    }, [rooms, filters]);

    const hasActiveFilters = filters.search.length > 0 || (filters.building && filters.building !== 'all') ||
                            !!filters.minCapacity;

    return { filteredRooms, hasActiveFilters };
}
