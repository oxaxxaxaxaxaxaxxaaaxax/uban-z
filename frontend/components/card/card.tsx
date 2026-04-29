'use client';

import Link from 'next/link';
import Image from 'next/image';
import Typography from '@mui/material/Typography';
import Paper from '@mui/material/Paper';
import Box from '@mui/material/Box';

import styles from './card.module.scss';

export interface CardProps {
    title: string;
    icon: string;
    href: string;
}

export default function ActionCard({title, icon, href}: CardProps) {
    return (
        <Link href={href} className={styles.cardLink}>
            <Paper className={styles.card} elevation={2}>
                <Box className={styles.cardIcon}>
                    <Image
                        src={icon}
                        alt=""
                        fill
                        sizes="100x" 
                        className={styles.iconImage}
                        aria-hidden="true" 
                    /> 
                </Box>
                <Typography variant="h6" className={styles.cardTitle}>
                    {title}
                </Typography>
            </Paper>
        </Link>
    );
}
