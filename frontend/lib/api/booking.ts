import { apiRequest } from "./client";
import type { components } from '@/types/booking';
import { logger } from '../logger';

type Room = components['schemas']['Room'];
type ScheduleItem = components['schemas']['ScheduleItem'];
type CreateBookingRequest = components['schemas']['CreateBookingRequest'];
type Booking = components['schemas']['Booking'];

export interface ScheduleImportStatus {
  status: 'pending' | 'running' | 'ready' | 'failed';
  error?: string;
  started_at?: string;
  completed_at?: string;
  stats?: {
    RoomsSeen: number;
    RoomsImported: number;
    LessonsSeen: number;
    LessonsExpanded: number;
    LessonsImported: number;
    LessonsSkipped: number;
  };
}

interface ConflictInfo {
  message: string;
}

export async function getRooms(): Promise<{
  success: boolean; rooms?: Room[]; error?: { status: number; message: string };
}> {
    const { data, error } = await apiRequest<Room[]>('/rooms', { method: 'GET' });
    if (error) {
        logger.error('getRoom error', error)
        return { success: false, error };
    }
  
     logger.info('getRoom successful');
    return { success: true, rooms: data || [] };
}

export async function getRoomSchedule(id: number): Promise<{
  success: boolean; schedule?: ScheduleItem[]; error?: { status: number; message: string };
}> {
  const { data, error } = await apiRequest<ScheduleItem[]>(`/rooms/${id}`, {
    method: 'GET',
  });

  if (error) {
    logger.error('getRoomSchedule error', error)
    return { success: false, error };
  }

  logger.info('getRoomSchedule successful');
  return { success: true, schedule: data || [] };
}

export async function getScheduleImportStatus(): Promise<{
  success: boolean; importStatus?: ScheduleImportStatus; error?: { status: number; message: string };
}> {
  const { data, error } = await apiRequest<ScheduleImportStatus>('/parser/status', {
    method: 'GET',
  });

  if (error) {
    logger.error('getScheduleImportStatus error', error)
    return { success: false, error };
  }

  logger.info('getScheduleImportStatus successful');
  return { success: true, importStatus: data };
}

export async function createBooking(id: number, startTime: string, endTime: string): Promise<{
  success: boolean; booking?: Booking; error?: { status: number; message: string; conflictInfo?: ConflictInfo };
}> {
  const { data, error } = await apiRequest<Booking>('/booking', {
    method: 'POST',
    body: JSON.stringify({
      room_id: id,
      start_time: startTime,
      end_time: endTime
    } as CreateBookingRequest),
  });

  if (error) {
    if (error.status === 409) {
      return {
        success: false,
        error: {
          ...error,
          conflictInfo: { message: 'Конфликт расписания' }
        }
      };
    }
    logger.error('createBooking error', error)
    return { success: false, error };
  }

  logger.info('createBooking successful');
  return { success: true, booking: data };
}


export async function cancelBooking(bookingId: number): Promise<{
  success: boolean; error?: { status: number; message: string };
}> {
  const { error } = await apiRequest<null>(`/booking/${bookingId}`, { method: 'DELETE' });

  if (error) {
      logger.error('cancelBooking error', error)
      return { 
          success: false, 
          error 
      };
  }
  logger.info('cancelBooking successful');
  return { success: true };
}


export async function getUserBookings(token?: string): Promise<{
  success: boolean; bookings?: Booking[]; error?: { status: number; message: string };
}> {
  const { data, error } = await apiRequest<Booking[]>('/booking/my', {
    method: 'GET',
    authToken: token,
  });
    
  if (error) {
    logger.error('getUserBookings error', error)
    return {
      success: false,
      error
    };
  }

  logger.info('getUserBookings successful');
  return {
      success: true,
      bookings: data || []
  };
}
