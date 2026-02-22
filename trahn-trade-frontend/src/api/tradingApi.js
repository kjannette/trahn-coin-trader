const API_BASE = 'http://localhost:3001/v1';

export const tradingApi = {

  getCurrentPrices: async () => {
    const response = await fetch(`${API_BASE}/prices/today`);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return response.json();
  },

  getCurrentTrades: async () => {
    const response = await fetch(`${API_BASE}/trades/today`);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return response.json();
  },

  getCurrentGrid: async () => {
    const response = await fetch(`${API_BASE}/grid/current`);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return response.json();
  },


  getLatestSR: async () => {
    const response = await fetch(`${API_BASE}/support-resistance/latest`);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return response.json();
  },

  getAvailableDays: async () => {
    const response = await fetch(`${API_BASE}/prices/days`);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return response.json();
  },

  getPricesByDay: async (date) => {
    const response = await fetch(`${API_BASE}/prices/day/${date}`);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return response.json();
  },

  getTradesByDay: async (date) => {
    const response = await fetch(`${API_BASE}/trades/day/${date}`);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return response.json();
  },

  getTradeStats: async () => {
    const response = await fetch(`${API_BASE}/trades/stats`);
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    return response.json();
  },
};

