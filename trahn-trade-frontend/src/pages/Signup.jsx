import { useState } from 'react';
import { Link } from 'react-router-dom';
import { addToWaitlist } from '../auth/firebase';
//a comment

export function Signup() {
    const [email, setEmail] = useState('');
    const [loading, setLoading] = useState(false);
    const [submitted, setSubmitted] = useState(false);
    const [error, setError] = useState('');
//
//
// 
    async function handleSubmit(e) {
        e.preventDefault();
        
        if (!email) {
            return setError('Please enter your email address');
        }

        try {
            setError('');
            setLoading(true);
            const result = await addToWaitlist(email);
            
            if (result.success) {
                setSubmitted(true);
            } else {
                setError(result.error || 'Failed to join waitlist. Please try again.');
            }
        } catch (err) {
            setError('Something went wrong. Please try again.');
        }
        setLoading(false);
    }
// some spacing 
//more
    return (
        <div className="auth-container">
            <div className="auth-card">
                <pre className="ascii-logo">{`
   ████████╗██████╗  █████╗ ██╗  ██╗███╗   ██╗
   ╚══██╔══╝██╔══██╗██╔══██╗██║  ██║████╗  ██║
      ██║   ██████╔╝███████║███████║██╔██╗ ██║
      ██║   ██╔══██╗██╔══██║██╔══██║██║╚██╗██║
      ██║   ██║  ██║██║  ██║██║  ██║██║ ╚████║
      ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝`}</pre>
                
                {submitted ? (
                    <>
                        <h2>Request Received</h2>
                        <div className="waitlist-message">
                            <p>
                                A message has been sent requesting authorization for signup. 
                                You will receive a response to your email when granted, or 
                                further information regarding the waitlist.
                            </p>
                            <p className="thank-you">Thank you for your interest!</p>
                        </div>
                        <p className="auth-link">
                            Already have an account? <Link to="/login">Log In</Link>
                        </p>
                    </>
                ) : (
                    <>
                        <h2>Join Waitlist</h2>
                        
                        {error && <div className="auth-error">{error}</div>}
                        
                        <form onSubmit={handleSubmit} className="auth-form">
                            <div className="form-group">
                                <label htmlFor="email">Email</label>
                                <input
                                    type="email"
                                    id="email"
                                    value={email}
                                    onChange={(e) => setEmail(e.target.value)}
                                    required
                                    placeholder="you@example.com"
                                />
                            </div>
                            
                            <button type="submit" className="auth-btn" disabled={loading}>
                                {loading ? 'Submitting...' : 'Request Access'}
                            </button>
                        </form>
                        
                        <p className="auth-link">
                            Already have an account? <Link to="/login">Log In</Link>
                        </p>
                    </>
                )}
            </div>
        </div>
    );
}
