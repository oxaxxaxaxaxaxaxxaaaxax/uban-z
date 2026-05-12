'use client';

import styles from './backButton.module.scss';

import { useRouter } from 'next/navigation';
import Button from '@mui/material/Button';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';

interface BackButtonProps {
    label?: string;
    fallback?: string;
}

export default function BackButton({label = 'Назад', fallback = '/'}: BackButtonProps) {
    const router = useRouter();

    const handleBack = () => {
        if (window.history.length > 1) {
            router.back();
        } else {
            router.push(fallback);
        }
    };

    return (
        <Button
            variant="text"
            color="inherit"
            startIcon={<ArrowBackIcon />}
            onClick={handleBack}
            className={styles.button}
        >
            {label}
        </Button>
    );
}
