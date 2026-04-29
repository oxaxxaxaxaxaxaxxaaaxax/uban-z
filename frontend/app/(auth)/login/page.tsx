'use client' 

import { useState } from "react";
import { useForm } from "react-hook-form";
import { useRouter } from 'next/navigation';

import TextField from "@mui/material/TextField";
import Paper from "@mui/material/Paper";
import Button from "@mui/material/Button";
import Alert from "@mui/material/Alert";

import IconButton from "@mui/material/IconButton";
import InputAdornment from "@mui/material/InputAdornment";
import Visibility from "@mui/icons-material/Visibility";     
import VisibilityOff from "@mui/icons-material/VisibilityOff";

import styles from './page.module.scss';

import { mockLogin } from '@/lib/testData';


interface LoginFormInputs {
    username: string,
    password: string,
};

export default function LoginPage() {
    const router = useRouter();
    const [showPassword, setShowPassword] = useState(false);

    const {register, handleSubmit, formState: { errors, isValid, isSubmitting, isDirty }, setError} = 
    useForm<LoginFormInputs>({
        mode: 'onChange',
        defaultValues: {
            username: '',
            password: '',
        },
    })

    const isButtonDisabled = (isDirty && !isValid) || isSubmitting

    const onSubmit = async (data: LoginFormInputs) => {
        // replace: fetch to API Gateway - login
        const result = await mockLogin(data.username, data.password);

        if (!result.success) {
            setError('root', {
                type: 'manual',
                message: 'Неверный логин или пароль',
            })
            return
        }

        router.push('/');
        router.refresh();
    }

    return (
        <main className={styles.container}>
            <Paper className={styles.card} elevation={2}>
                <div className={styles.logoWrapper}>
                    <img
                        src="/nsu-logo.png"
                        alt="Логотип НГУ"
                        className={styles.logoImage}
                    />
                </div>

                <div className={styles.content}>
                    <h1 className={styles.title}>
                        NSU ID (Вход под университетским аккаунтом)
                    </h1>

                    {errors.root && (
                        <Alert
                            severity="error"
                            className={styles.alert}
                        >
                            {errors.root.message}
                        </Alert>
                    )}

                    <form onSubmit={handleSubmit(onSubmit)}
                        noValidate
                        suppressHydrationWarning>
                        <TextField
                            className={styles.field}
                            variant="standard"
                            label="Username"
                            error={Boolean(errors.username?.message)}
                            helperText={errors.username?.message}
                            {...register(`username`, {
                                required: 'Введите логин',
                                pattern: {
                                    value: /^[a-z]\.[a-z]{2,}\d{0,2}$/,
                                    message: 'Некорректный формат логина'
                                },
                             })}
                            fullWidth
                        />

                        <TextField className={styles.field}
                            variant="standard"
                            label="Password"
                            type={showPassword ? 'text' : 'password'}
                            error={Boolean(errors.password?.message)}
                            helperText={errors.password?.message}
                            {...register(`password`, {
                                required: 'Введите пароль',
                                minLength: {
                                    value: 4,
                                    message: 'Минимум 4 символа'
                                }
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

                        <Button className={styles.submitButton}
                            type="submit"
                            variant="contained"
                            size="large"
                            disabled={isButtonDisabled}
                            suppressHydrationWarning
                            fullWidth>
                            {isSubmitting ? 'Sign in...' : 'Sign in'}
                        </Button>
                    </form>
                </div>
            </Paper>
        </main >
    );
}
