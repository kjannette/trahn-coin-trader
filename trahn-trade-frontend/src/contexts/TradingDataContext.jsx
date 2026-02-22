import { createContext, useContext, useReducer, useEffect } from 'react';
import { useQuery } from '@tanstack/react-query';
import { tradingApi } from '../api/tradingApi';

const TradingDataContext = createContext(undefined);

const initialState = {
    currentDay: null,
    availableDays: [],
    currentDayIndex: 0,
    prices: [],
    trades: [],
    grid: [],
    supportResistance: null,
    isLive: true,
    connectionStatus: 'connecting',
};

function reducer(state, action) {
    switch (action.type) {
        case 'UPDATE_CURRENT_DAY':
            const dayIndex = action.availableDays.length > 0 ? action.availableDays.length - 1 : 0;
            return {
                ...state,
                prices: action.prices || [],
                trades: action.trades || [],
                grid: action.grid || [],
                availableDays: action.availableDays || [],
                currentDay: action.currentDay,
                currentDayIndex: dayIndex,
                isLive: true,
                connectionStatus: 'connected',
            };
        
        case 'LOAD_HISTORICAL_DAY':
            return {
                ...state,
                prices: action.prices,
                trades: action.trades,
                currentDayIndex: action.index,
                isLive: false,
            };
        
        case 'SET_CONNECTION_STATUS':
            return {
                ...state,
                connectionStatus: action.status,
            };
        
        case 'SWITCH_TO_LIVE':
            return {
                ...state,
                isLive: true,
                currentDayIndex: state.availableDays.length - 1,
            };
        
        default:
            return state;
    }
}

export function TradingDataProvider({ children }) {
    const [state, dispatch] = useReducer(reducer, initialState);
    
    // React Query: Poll current day data every 1 minute
    const { data: currentData, error, isLoading } = useQuery({
        queryKey: ['currentDay'],
        queryFn: async () => {
            const [prices, trades, grid, days] = await Promise.all([
                tradingApi.getCurrentPrices(),
                tradingApi.getCurrentTrades(),
                tradingApi.getCurrentGrid(),
                tradingApi.getAvailableDays(),
            ]);
            
            return { 
                prices, 
                trades, 
                grid: grid.grid || [], 
                availableDays: days,
                currentDay: days[days.length - 1] || null,
            };
        },
        refetchInterval: 60000, // Poll every 1 minute
        staleTime: 30000, // Data fresh for 30 seconds
        retry: 3,
        retryDelay: 2000,
    });
    
    // Update context when live data changes
    useEffect(() => {
        if (currentData) {
            dispatch({ 
                type: 'UPDATE_CURRENT_DAY', 
                ...currentData 
            });
        }
    }, [currentData]);
    
    // Handle connection status
    useEffect(() => {
        if (error) {
            dispatch({ type: 'SET_CONNECTION_STATUS', status: 'error' });
        } else if (isLoading) {
            dispatch({ type: 'SET_CONNECTION_STATUS', status: 'connecting' });
        } else {
            dispatch({ type: 'SET_CONNECTION_STATUS', status: 'connected' });
        }
    }, [error, isLoading]);
    
    return (
        <TradingDataContext.Provider value={{ state, dispatch }}>
            {children}
        </TradingDataContext.Provider>
    );
}

export function useTradingData() {
    const context = useContext(TradingDataContext);
    if (!context) {
        throw new Error('useTradingData must be used within TradingDataProvider');
    }
    return context;
}

