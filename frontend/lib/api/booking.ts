import { apiRequest } from "./client";
import type { components } from '@/types/booking';

type Room = components['schemas']['Room'];
type ScheduleItem = components['schemas']['ScheduleItem'];
type CreateBookingRequest = components['schemas']['CreateBookingRequest'];
type Booking = components['schemas']['Booking'];


export async function getRooms(): Promise<{
  success: boolean; rooms?: Room[]; error?: { status: number; message: string };
}> {
    const response = await fetch('http://localhost:3000/testData/rooms.json');
    const mockData = await response.json();
    return { 
        success: true, 
        rooms: mockData as Room[] 
    };

    // const { data, error } = await apiRequest<Room[]>('/rooms', { method: 'GET' });
    // if (error) {
    //     return { success: false, error };
    // }

    // return { success: true, rooms: data || [] };
}

export async function getRoomSchedule(id: number): Promise<{
  success: boolean; schedule?: ScheduleItem[]; error?: { status: number; message: string };
}> {
    const response = await fetch('http://localhost:3000/testData/schedules.json');
    const mockData = await response.json();
    return { 
        success: true, 
        schedule: mockData as ScheduleItem[] 
    };

//   const { data, error } = await apiRequest<ScheduleItem[]>(`/rooms/${id}`, {
//     method: 'GET',
//   });

//   if (error) {
//     return { success: false, error };
//   }

//   return { success: true, schedule: data || [] };
}

export async function createBooking(id: number, startTime: string, endTime: string): Promise<{
  success: boolean; booking?: Booking; error?: { status: number; message: string; conflictInfo?: any };
}> {

    // Симуляция конфликта
    if (id === 2 && startTime.includes('T14:30')) {
        return {
            success: false,
            error: {
                status: 409,
                message: 'Аудитория уже занята в это время',
                conflictInfo: {
                    type: 'Лекция: Мат Анализ',
                    teacher: 'Васечкин В',
                    groups: ["25425", "25426", "25427", "25428", "25429", "25430"]
                }
            }
        };
    }
    
    return {
        success: true,
        booking: {
            id: Date.now(),
            room_id: id,
            start_time: startTime,
            end_time: endTime
        }
    };

//   const { data, error } = await apiRequest<Booking>('/booking', {
//     method: 'POST',
//     body: JSON.stringify({
//       room_id: roomId,
//       start_time: startTime,
//       end_time: endTime
//     } as CreateBookingRequest),
//   });

//   if (error) {
//     if (error.status === 409) {
//       return {
//         success: false,
//         error: {
//           ...error,
//           conflictInfo: { message: 'Конфликт расписания' }
//         }
//       };
//     }
//     return { success: false, error };
//   }

//   return { success: true, booking: data };
}


export async function cancelBooking(bookingId: number): Promise<{
  success: boolean; error?: { status: number; message: string };
}> {

    return { success: true };

//   const { error } = await apiRequest<null>(`/booking/${bookingId}`, { method: 'DELETE' });

//   if (error) {
//       return { 
//           success: false, 
//           error 
//       };
//   }

//   return { success: true };
}


export async function getUserBookings(): Promise<{
  success: boolean; bookings?: Booking[]; error?: { status: number; message: string };
}> {
    const response = await fetch('http://localhost:3000/testData/user-bookings.json');
    const mockData = await response.json();
    return {
        success: true,
        bookings: mockData
    };
  
    // const { data, error } = await apiRequest<Booking[]>('/mybooking', { method: 'GET' });
    
    // if (error) return {
    //     success: false,
    //     error
    // };
    // return {
    //     success: true,
    //     bookings: data || []
    // };
}

