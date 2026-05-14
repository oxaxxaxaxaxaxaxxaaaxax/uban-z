'use client';

import { useState } from "react";
import { useForm } from "react-hook-form";
import { useRouter } from 'next/navigation';
import {
    Paper, TextField, Button, Alert, IconButton,
    InputAdornment
} from "@mui/material";
import Visibility from "@mui/icons-material/Visibility";
import VisibilityOff from "@mui/icons-material/VisibilityOff";
import { login } from '@/lib/api/auth';
import styles from './loginForm.module.scss';

interface LoginFormInputs {
    username: string;
    password: string;
}

export default function LoginForm() {
    const router = useRouter();
    const [showPassword, setShowPassword] = useState(false);

    const { register, handleSubmit, formState: { errors, isValid, isSubmitting, isDirty }, setError } =
        useForm<LoginFormInputs>({
            mode: 'onChange',
            defaultValues: { username: '', password: '' },
        });

    const isButtonDisabled = (isDirty && !isValid) || isSubmitting;

    const onSubmit = async (data: LoginFormInputs) => {
        const result = await login(data.username, data.password);

        if (!result.success) {
            let message = 'Неверный логин или пароль';
            if (result.error?.status === 500) message = 'Сервер временно недоступен';
            if (result.error?.status === 0) message = 'Нет соединения с интернетом';

            setError('root', { type: 'manual', message });
            return;
        }

        if (result.token) {
            document.cookie = `session_token=${result.token}; path=/; max-age=3600; SameSite=Lax`;
        }
        router.push('/');
    };

    return (
        <Paper className={styles.card} elevation={2}>
            <div className={styles.logoWrapper}>
                <img src="/nsu-logo.png" alt="Логотип НГУ" className={styles.logoImage} />
            </div>

            <div className={styles.content}>
                <h1 className={styles.title}>
                    NSU ID (Вход под университетским аккаунтом)
                </h1>

                {errors.root && (
                    <Alert severity="error" className={styles.alert}>
                        {errors.root.message}
                    </Alert>
                )}

                <form onSubmit={handleSubmit(onSubmit)} noValidate>
                    <TextField
                        className={styles.field}
                        variant="standard"
                        label="Username"
                        error={Boolean(errors.username?.message)}
                        helperText={errors.username?.message}
                        {...register('username', {
                            required: 'Введите логин',
                            pattern: {
                                value: /^[a-z]\.[a-z]{2,}\d{0,2}$/,
                                message: 'Некорректный формат логина'
                            }
                        })}
                        fullWidth
                    />

                    <TextField
                        className={styles.field}
                        variant="standard"
                        label="Password"
                        type={showPassword ? 'text' : 'password'}
                        error={Boolean(errors.password?.message)}
                        helperText={errors.password?.message}
                        {...register('password', {
                            required: 'Введите пароль',
                            minLength: { value: 4, message: 'Минимум 4 символа' }
                        })}
                        fullWidth
                        slotProps={{
                            input: {
                                endAdornment: (
                                    <InputAdornment position="end">
                                        <IconButton
                                            onClick={() => setShowPassword(!showPassword)}
                                            edge="end"
                                            tabIndex={-1}
                                        >
                                            {showPassword ? <Visibility /> : <VisibilityOff />}
                                        </IconButton>
                                    </InputAdornment>
                                ),
                            },
                        }}
                    />

                    <Button
                        className={styles.submitButton}
                        type="submit"
                        variant="contained"
                        size="large"
                        disabled={isButtonDisabled}
                        fullWidth
                    >
                        {isSubmitting ? 'Sign in...' : 'Sign in'}
                    </Button>
                </form>
            </div>
        </Paper>
    );
}
