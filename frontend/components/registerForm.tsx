'use client';

import { useState } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { useRouter } from 'next/navigation';
import {
    Paper, TextField, Button, Alert, MenuItem, Link,
    IconButton, InputAdornment
} from '@mui/material';
import Visibility from '@mui/icons-material/Visibility';
import VisibilityOff from '@mui/icons-material/VisibilityOff';
import { register as registerUser } from '@/lib/api/auth';
import styles from './authForm.module.scss';

interface RegisterFormInputs {
    login: string;
    password: string;
    confirmPassword: string;
    role: string;
}

export default function RegisterForm() {
    const router = useRouter();
    const [showPassword, setShowPassword] = useState(false);
    const [showConfirmPassword, setShowConfirmPassword] = useState(false);

    const { register, control, handleSubmit, formState: { errors, isValid, isSubmitting }, setError, watch } =
        useForm<RegisterFormInputs>({
            mode: 'onChange',
            defaultValues: { login: '', password: '', confirmPassword: '', role: '' },
        });

    const password = watch('password');

    const onSubmit = async (data: RegisterFormInputs) => {
        if (data.password !== data.confirmPassword) {
            setError('confirmPassword', { type: 'manual', message: 'Пароли не совпадают' });
            return;
        }

        const result = await registerUser(data.login, data.password, data.role);

        if (!result.success) {
            let message = 'Ошибка регистрации. Проверьте данные.';

            if (result.error?.message) {
                message = result.error.message;
            } else if (result.error?.status === 500) {
                message = 'Сервер временно недоступен';
            } else if (result.error?.status === 0) {
                message = 'Нет соединения с интернетом';
            }

            setError('root', { type: 'manual', message });
            return;
        }

        router.push('/login');
    };

    return (
        <Paper className={styles.card} elevation={2}>
            <div className={styles.logoWrapper}>
                <img src="/nsu-logo.png" alt="Логотип НГУ" className={styles.logoImage} />
            </div>

            <div className={styles.content}>
                <h1 className={styles.title}>
                    Регистрация под университетским аккаунтом
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
                        label="Логин"
                        error={Boolean(errors.login?.message)}
                        helperText={errors.login?.message}
                        {...register('login', {
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
                        label="Пароль"
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

                    <TextField
                        className={styles.field}
                        variant="standard"
                        label="Подтвердите пароль"
                        type={showConfirmPassword ? 'text' : 'password'}
                        error={Boolean(errors.confirmPassword?.message)}
                        helperText={errors.confirmPassword?.message}
                        {...register('confirmPassword', {
                            required: 'Подтвердите пароль',
                            validate: value => value === password || 'Пароли не совпадают'
                        })}
                        fullWidth
                        slotProps={{
                            input: {
                                endAdornment: (
                                    <InputAdornment position="end">
                                        <IconButton
                                            onClick={() => setShowConfirmPassword(!showConfirmPassword)}
                                            edge="end"
                                            tabIndex={-1}
                                        >
                                            {showConfirmPassword ? <Visibility /> : <VisibilityOff />}
                                        </IconButton>
                                    </InputAdornment>
                                ),
                            },
                        }}
                    />

                    <Controller
                        name="role"
                        control={control}
                        rules={{ required: 'Выберите роль' }}
                        render={({ field }) => (
                            <TextField
                                className={styles.field}
                                variant="standard"
                                select
                                label="Роль"
                                error={Boolean(errors.role?.message)}
                                helperText={errors.role?.message}
                                fullWidth
                                {...field} 
                            >
                                <MenuItem value="student_b">Бакалавр</MenuItem>
                                <MenuItem value="student_m">Магистрант</MenuItem>
                                <MenuItem value="student_a">Аспирант</MenuItem>
                                <MenuItem value="teacher">Преподаватель</MenuItem>
                                <MenuItem value="admin">Администратор</MenuItem>             
                            </TextField>
                        )}
                    />

                    <Button
                        className={styles.submitButton}
                        type="submit"
                        variant="contained"
                        size="large"
                        disabled={isSubmitting || !isValid}
                        fullWidth
                    >
                        {isSubmitting ? 'Регистрация...' : 'Зарегистрироваться'}
                    </Button>

                    <div className={styles.footer}>
                        Уже есть аккаунт?{' '}
                        <Link href="/login" className={styles.link}>
                            Войти
                        </Link>
                    </div>
                </form>
            </div>
        </Paper>
    );
}
