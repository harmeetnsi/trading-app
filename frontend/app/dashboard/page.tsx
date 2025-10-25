
'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { RefreshCw } from 'lucide-react';
import NavBar from '@/components/ui/NavBar';
import PortfolioCard from '@/components/dashboard/PortfolioCard';
import PositionsList from '@/components/dashboard/PositionsList';
import RecentTrades from '@/components/dashboard/RecentTrades';
import { api } from '@/lib/api';
import { useAuthStore } from '@/lib/store';

export default function DashboardPage() {
  const router = useRouter();
  const setUser = useAuthStore((state) => state.setUser);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [portfolio, setPortfolio] = useState<any>(null);
  const [trades, setTrades] = useState<any[]>([]);

  useEffect(() => {
    const init = async () => {
      try {
        const profileResponse = await api.getProfile();
        if (profileResponse.success) {
          setUser(profileResponse.data);
        }
        await loadData();
      } catch (error) {
        router.push('/auth/login');
      }
    };

    init();
  }, []);

  const loadData = async () => {
    try {
      setLoading(true);

      // Load portfolio data
      try {
        const portfolioResponse = await api.getPortfolio();
        if (portfolioResponse.success) {
          setPortfolio(portfolioResponse.data);
        }
      } catch (error) {
        console.error('Failed to load portfolio:', error);
        // Set default portfolio for demo
        setPortfolio({
          total_value: 0,
          cash: 0,
          positions_value: 0,
          today_pnl: 0,
          total_pnl: 0,
          positions: [],
        });
      }

      // Load recent trades
      try {
        const tradesResponse = await api.getTrades(10);
        if (tradesResponse.success) {
          setTrades(tradesResponse.data || []);
        }
      } catch (error) {
        console.error('Failed to load trades:', error);
        setTrades([]);
      }
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleRefresh = async () => {
    setRefreshing(true);
    await loadData();
    setRefreshing(false);
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50">
        <NavBar />
        <div className="flex items-center justify-center h-[calc(100vh-64px)]">
          <div className="spinner border-primary-600"></div>
        </div>
      </div>
    );
  }

  const totalPnLPercent = portfolio?.total_value
    ? (portfolio.total_pnl / portfolio.total_value) * 100
    : 0;

  return (
    <div className="min-h-screen bg-gray-50">
      <NavBar />

      <div className="max-w-7xl mx-auto px-4 py-6">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
          <button
            onClick={handleRefresh}
            disabled={refreshing}
            className="btn btn-secondary flex items-center gap-2"
          >
            <RefreshCw className={`w-4 h-4 ${refreshing ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>

        {/* Portfolio Overview */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          <PortfolioCard
            title="Total Value"
            value={`₹${portfolio?.total_value?.toFixed(2) || '0.00'}`}
            subtitle="Portfolio value"
          />
          <PortfolioCard
            title="Available Cash"
            value={`₹${portfolio?.cash?.toFixed(2) || '0.00'}`}
            subtitle="Cash balance"
          />
          <PortfolioCard
            title="Today's P&L"
            value={`₹${portfolio?.today_pnl?.toFixed(2) || '0.00'}`}
            variant={portfolio?.today_pnl >= 0 ? 'success' : 'error'}
            subtitle="Today's profit/loss"
          />
          <PortfolioCard
            title="Total P&L"
            value={`₹${portfolio?.total_pnl?.toFixed(2) || '0.00'}`}
            change={totalPnLPercent}
            variant={portfolio?.total_pnl >= 0 ? 'success' : 'error'}
            subtitle="Overall profit/loss"
          />
        </div>

        {/* Positions and Trades */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <PositionsList positions={portfolio?.positions || []} />
          <RecentTrades trades={trades} />
        </div>

        {/* Quick Actions */}
        <div className="mt-6 card">
          <h2 className="text-lg font-semibold mb-4">Quick Actions</h2>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
            <button
              onClick={() => router.push('/chat')}
              className="btn btn-primary"
            >
              Ask AI
            </button>
            <button
              onClick={() => router.push('/strategies')}
              className="btn btn-secondary"
            >
              Strategies
            </button>
            <button
              onClick={() => alert('Coming soon!')}
              className="btn btn-secondary"
            >
              Place Order
            </button>
            <button
              onClick={() => alert('Coming soon!')}
              className="btn btn-secondary"
            >
              Analysis
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
