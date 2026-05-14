export const DASHBOARD_CARDS = [
  {
    title: 'Список аудиторий',
    icon: '/rooms_list.png',
    href: '/rooms',
  },
  {
    title: 'Расписание аудиторий',
    icon: '/timetable.png',
    href: '/rooms/timetable',
  },
  {
    title: 'Создать бронь',
    icon: '/create_booking.png',
    href: '/booking/create',
  },
  {
    title: 'Отменить бронь',
    icon: '/delete_booking.png',
    href: '/booking/delete',
  },
] as const;
