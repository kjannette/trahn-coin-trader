import { useQuery } from '@tanstack/react-query';
import { useTradingData } from '../contexts/TradingDataContext';
import { tradingApi } from '../api/tradingApi';

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

export function useCarousel() {
    const { state, dispatch } = useTradingData();
    
    const navigateTo = (index) => {
        if (index < 0 || index >= state.availableDays.length) {
            return;
        }
        
        if (index === state.availableDays.length - 1) {
            dispatch({ type: 'SWITCH_TO_LIVE' });
        } else {
            const date = state.availableDays[index];
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

