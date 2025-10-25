
import axios, { AxiosInstance } from 'axios';

const API_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const WS_URL = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080/ws';

class APIClient {
  private client: AxiosInstance;

  constructor() {
    this.client = axios.create({
      baseURL: API_URL,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add auth token to requests
    this.client.interceptors.request.use((config) => {
      const token = this.getToken();
      if (token) {
        config.headers.Authorization = `Bearer ${token}`;
      }
      return config;
    });

    // Handle auth errors
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response?.status === 401) {
          this.clearToken();
          if (typeof window !== 'undefined') {
            window.location.href = '/auth/login';
          }
        }
        return Promise.reject(error);
      }
    );
  }

  // Token management
  getToken(): string | null {
    if (typeof window === 'undefined') return null;
    return localStorage.getItem('token');
  }

  setToken(token: string) {
    if (typeof window === 'undefined') return;
    localStorage.setItem('token', token);
  }

  clearToken() {
    if (typeof window === 'undefined') return;
    localStorage.removeItem('token');
  }

  // Auth endpoints
  async register(username: string, password: string) {
    const response = await this.client.post('/api/auth/register', {
      username,
      password,
    });
    return response.data;
  }

  async login(username: string, password: string) {
    const response = await this.client.post('/api/auth/login', {
      username,
      password,
    });
    if (response.data.success) {
      this.setToken(response.data.data.token);
    }
    return response.data;
  }

  async logout() {
    try {
      await this.client.post('/api/auth/logout');
    } finally {
      this.clearToken();
    }
  }

  async getProfile() {
    const response = await this.client.get('/api/auth/profile');
    return response.data;
  }

  // Chat endpoints
  async getMessages(limit = 50) {
    const response = await this.client.get(`/api/chat/messages?limit=${limit}`);
    return response.data;
  }

  async sendMessage(content: string, fileId?: number) {
    const response = await this.client.post('/api/chat/send', {
      content,
      file_id: fileId,
    });
    return response.data;
  }

  // File endpoints
  async uploadFile(file: File) {
    const formData = new FormData();
    formData.append('file', file);

    const response = await this.client.post('/api/files/upload', formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });
    return response.data;
  }

  async getFiles() {
    const response = await this.client.get('/api/files');
    return response.data;
  }

  async getFile(id: number) {
    const response = await this.client.get(`/api/files/get?id=${id}`);
    return response.data;
  }

  // Strategy endpoints
  async getStrategies() {
    const response = await this.client.get('/api/strategies');
    return response.data;
  }

  async getStrategy(id: number) {
    const response = await this.client.get(`/api/strategies/get?id=${id}`);
    return response.data;
  }

  async createStrategy(data: {
    name: string;
    description: string;
    file_id: number;
    code: string;
  }) {
    const response = await this.client.post('/api/strategies/create', data);
    return response.data;
  }

  async updateStrategyStatus(id: number, status: string) {
    const response = await this.client.put(`/api/strategies/status?id=${id}`, {
      status,
    });
    return response.data;
  }

  async getBacktestResults(strategyId: number) {
    const response = await this.client.get(
      `/api/strategies/backtest-results?strategy_id=${strategyId}`
    );
    return response.data;
  }

  // Backtest endpoints
  async runBacktest(data: {
    strategy_id: number;
    start_date: string;
    end_date: string;
    initial_capital: number;
    symbol: string;
    exchange: string;
  }) {
    const response = await this.client.post('/api/backtest/run', data);
    return response.data;
  }

  // Trade endpoints
  async getTrades(limit = 50) {
    const response = await this.client.get(`/api/trades?limit=${limit}`);
    return response.data;
  }

  // Portfolio endpoints
  async getPortfolio() {
    const response = await this.client.get('/api/portfolio');
    return response.data;
  }

  async getPositions() {
    const response = await this.client.get('/api/portfolio/positions');
    return response.data;
  }

  async getHoldings() {
    const response = await this.client.get('/api/portfolio/holdings');
    return response.data;
  }

  async placeOrder(order: {
    symbol: string;
    exchange: string;
    action: string;
    quantity: number;
    price?: number;
    order_type: string;
    product: string;
    pricetype: string;
  }) {
    const response = await this.client.post('/api/portfolio/order', order);
    return response.data;
  }

  async getQuote(symbol: string, exchange: string) {
    const response = await this.client.get(
      `/api/portfolio/quote?symbol=${symbol}&exchange=${exchange}`
    );
    return response.data;
  }

  // WebSocket connection
  createWebSocket(): WebSocket | null {
    const token = this.getToken();
    if (!token) return null;

    const ws = new WebSocket(`${WS_URL}?token=${token}`);
    return ws;
  }
}

export const api = new APIClient();
export { WS_URL };
