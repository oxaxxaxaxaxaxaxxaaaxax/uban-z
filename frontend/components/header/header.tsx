'use client';

import Link from 'next/link';
import { useRouter } from 'next/navigation';
import Button from '@mui/material/Button';

import { mockLogout } from '@/lib/testData';

import styles from './header.module.scss';

interface HeaderProps {
    fullName?: string;
}

export default function Header({ fullName }: HeaderProps) {
    const router = useRouter();

    const handleLogout = () => {
        if (window.confirm('Вы уверены, что хотите выйти?')) {
            // replace: fetch to API Gateway - logout
            mockLogout();
            router.push('/login');
            router.refresh();
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
                    <span className={styles.userName}>{fullName}</span>
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
