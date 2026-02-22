import { useState } from 'react';
import { useNavigate, Link, useLocation } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';

export function Login() {
    const [email, setEmail] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const { login } = useAuth();
    const navigate = useNavigate();
    const location = useLocation();
    const successMessage = location.state?.message;

    async function handleSubmit(e) {
        e.preventDefault();

        try {
            setError('');
            setLoading(true);
            await login(email, password);
            navigate('/');
        } catch (err) {
            setError(err.message || 'Failed to log in');
        }
        setLoading(false);
    }

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
                
                <h2>Welcome Back</h2>
                
                {successMessage && <div className="auth-success">{successMessage}</div>}
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
                    
                    <div className="form-group">
                        <label htmlFor="password">Password</label>
                        <input
                            type="password"
                            id="password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                            required
                            placeholder="••••••••"
                        />
                    </div>
                    
                    <button type="submit" className="auth-btn" disabled={loading}>
                        {loading ? 'Logging In...' : 'Log In'}
                    </button>
                </form>
                
                <p className="auth-link">
                    Don't have an account? <Link to="/signup">Sign Up</Link>
                </p>
            </div>
        </div>
    );
}

