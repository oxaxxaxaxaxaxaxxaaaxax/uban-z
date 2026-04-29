import Header from '@/components/header';

import styles from './page.module.scss'

export default async function Rooms() {
    
    return (
        <main className={styles.container}>
            <div className={styles.content}>
                <Header fullName="Name"/>
            </div>
        </main>
    );
}
