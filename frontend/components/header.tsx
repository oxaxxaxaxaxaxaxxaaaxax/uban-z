'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import Button from '@mui/material/Button';

import { logout } from '@/lib/api/auth';

import styles from './header.module.scss';

interface HeaderProps {
    fullname?: string;
}

export default function Header({ fullname }: HeaderProps) {
    const router = useRouter();

    const handleLogout = () => {
        if (window.confirm('Вы уверены, что хотите выйти?')) {
            logout();
            router.push('/login');
        }
    };

    return (
        <div className={styles.root}>
            <div className={styles.inner}>
                <Link href="/" className={styles.logo}>
                    <img
                        src="/nsu-logo-2.png"
                        alt="НГУ"
                        className={styles.logoImage}
                    />
                </Link>

                <div className={styles.buttons}>
                    <span className={styles.userName}>{fullname}</span>
                    <Button
                        className={styles.logoutButton}
                        onClick={handleLogout}
                        variant="contained"
                    >
                        Выйти
                    </Button>
                </div>
            </div>
        </div>
    );
}
