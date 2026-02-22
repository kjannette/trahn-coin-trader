/**
 * Trading API Client
 * Communicates with backend REST API
 */

const API_BASE = 'http://localhost:3001/api';

export const tradingApi = {
    /**
     * Get current trading day prices
     */
    getCurrentPrices: async () => {
        const response = await fetch(`${API_BASE}/prices/today`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.json();
    },

    /**
     * Get current trading day trades
     */
    getCurrentTrades: async () => {
        const response = await fetch(`${API_BASE}/trades/today`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.json();
    },

    /**
     * Get current grid state
     */
    getCurrentGrid: async () => {
        const response = await fetch(`${API_BASE}/grid/current`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.json();
    },

    /**
     * Get latest support/resistance levels
     */
    getLatestSR: async () => {
        const response = await fetch(`${API_BASE}/support-resistance/latest`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.json();
    },

    /**
     * Get available trading days
     */
    getAvailableDays: async () => {
        const response = await fetch(`${API_BASE}/prices/days`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.json();
    },

    /**
     * Get historical prices for a specific day
     */
    getPricesByDay: async (date) => {
        const response = await fetch(`${API_BASE}/prices/day/${date}`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.json();
    },

    /**
     * Get historical trades for a specific day
     */
    getTradesByDay: async (date) => {
        const response = await fetch(`${API_BASE}/trades/day/${date}`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.json();
    },

    /**
     * Get trade statistics
     */
    getTradeStats: async () => {
        const response = await fetch(`${API_BASE}/trades/stats`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return response.json();
    },
};

