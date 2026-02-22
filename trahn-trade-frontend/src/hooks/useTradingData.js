/**
 * Custom hooks for trading data management
 * Handles carousel navigation and historical day loading
 */

import { useQuery } from '@tanstack/react-query';
import { useTradingData } from '../contexts/TradingDataContext';
import { tradingApi } from '../api/tradingApi';

/**
 * Hook for loading historical day data (lazy load)
 */
export function useHistoricalDay(date, enabled = false) {
    return useQuery({
        queryKey: ['historicalDay', date],
        queryFn: async () => {
            const [prices, trades] = await Promise.all([
                tradingApi.getPricesByDay(date),
                tradingApi.getTradesByDay(date),
            ]);
            return { prices, trades };
        },
        enabled: enabled && !!date,
        staleTime: Infinity, // Historical data never changes
        cacheTime: 600000, // Keep in cache for 10 minutes
    });
}

/**
 * Hook for carousel navigation
 */
export function useCarousel() {
    const { state, dispatch } = useTradingData();
    
    const navigateTo = (index) => {
        if (index < 0 || index >= state.availableDays.length) {
            return;
        }
        
        if (index === state.availableDays.length - 1) {
            // Navigate to live (current) day
            dispatch({ type: 'SWITCH_TO_LIVE' });
        } else {
            // Load historical day
            const date = state.availableDays[index];
            // Historical data will be loaded via useHistoricalDay in Dashboard
            dispatch({ 
                type: 'LOAD_HISTORICAL_DAY', 
                prices: [], 
                trades: [], 
                index 
            });
        }
    };
    
    const navigatePrev = () => {
        if (state.currentDayIndex > 0) {
            navigateTo(state.currentDayIndex - 1);
        }
    };
    
    const navigateNext = () => {
        if (state.currentDayIndex < state.availableDays.length - 1) {
            navigateTo(state.currentDayIndex + 1);
        }
    };
    
    return {
        currentDayIndex: state.currentDayIndex,
        availableDays: state.availableDays,
        currentDay: state.availableDays[state.currentDayIndex],
        canGoPrev: state.currentDayIndex > 0,
        canGoNext: state.currentDayIndex < state.availableDays.length - 1,
        navigatePrev,
        navigateNext,
        navigateTo,
        isLive: state.isLive,
    };
}

