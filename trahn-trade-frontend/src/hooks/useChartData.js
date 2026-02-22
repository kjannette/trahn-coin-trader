/**
 * DEPRECATED - Replaced by TradingDataContext + React Query
 * 
 * This file is kept for reference but no longer used.
 * The new implementation:
 * - Uses React Query for polling
 * - Uses Context for state management
 * - Calls REST API instead of JSON files
 * 
 * See:
 * - contexts/TradingDataContext.jsx
 * - hooks/useTradingData.js
 * - api/tradingApi.js
 */

export function useChartData() {
    throw new Error('useChartData is deprecated. Use useTradingData from TradingDataContext instead.');
}
