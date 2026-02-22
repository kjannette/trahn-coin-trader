import { initializeApp } from 'firebase/app';
import { getAuth } from 'firebase/auth';
import { getFirestore, collection, addDoc, serverTimestamp } from 'firebase/firestore';
import { firebaseConfig } from './secrets';

const app = initializeApp(firebaseConfig);
export const auth = getAuth(app);
export const db = getFirestore(app);

// Add email to waitlist collection
export async function addToWaitlist(email) {
    try {
        await addDoc(collection(db, 'trahn-trade-signups'), {
            email: email,
            createdAt: serverTimestamp(),
            status: 'pending'
        });
        return { success: true };
    } catch (error) {
        console.error('Error adding to waitlist:', error);
        return { success: false, error: error.message };
    }
}

export default app;

