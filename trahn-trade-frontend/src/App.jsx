import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { TradingDataProvider } from './contexts/TradingDataContext';
import { ProtectedRoute } from './components/ProtectedRoute';
import { Dashboard } from './pages/Dashboard';
import { Login } from './pages/Login';
import { Signup } from './pages/Signup';
import './App.css';

// Create React Query client
const queryClient = new QueryClient({
    defaultOptions: {
        queries: {
            refetchOnWindowFocus: false,
            retry: 3,
            retryDelay: 2000,
            staleTime: 30000,
        },
    },
});

// Redirect authenticated users away from auth pages
function PublicRoute({ children }) {
    const { currentUser } = useAuth();
    
    if (currentUser) {
        return <Navigate to="/" replace />;
    }
    
    return children;
}

function AppRoutes() {
    return (
        <Routes>
            <Route 
                path="/" 
                element={
                    <ProtectedRoute>
                        <Dashboard />
                    </ProtectedRoute>
                } 
            />
            <Route 
                path="/login" 
                element={
                    <PublicRoute>
                        <Login />
                    </PublicRoute>
                } 
            />
            <Route 
                path="/signup" 
                element={
                    <PublicRoute>
                        <Signup />
                    </PublicRoute>
                } 
            />
            <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
    );
}

function App() {
    return (
        <QueryClientProvider client={queryClient}>
            <Router>
                <AuthProvider>
                    <TradingDataProvider>
                        <AppRoutes />
                    </TradingDataProvider>
                </AuthProvider>
            </Router>
        </QueryClientProvider>
    );
}

export default App;
