
'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Plus, TrendingUp } from 'lucide-react';
import NavBar from '@/components/ui/NavBar';
import StrategyCard from '@/components/strategies/StrategyCard';
import BacktestModal from '@/components/strategies/BacktestModal';
import BacktestResults from '@/components/strategies/BacktestResults';
import { api } from '@/lib/api';

export default function StrategiesPage() {
  const router = useRouter();
  const [loading, setLoading] = useState(true);
  const [strategies, setStrategies] = useState<any[]>([]);
  const [selectedStrategy, setSelectedStrategy] = useState<any>(null);
  const [showBacktestModal, setShowBacktestModal] = useState(false);
  const [backtestResults, setBacktestResults] = useState<any[]>([]);
  const [runningBacktest, setRunningBacktest] = useState(false);

  useEffect(() => {
    const init = async () => {
      try {
        await api.getProfile();
        await loadStrategies();
      } catch (error) {
        router.push('/auth/login');
      }
    };

    init();
  }, []);

  const loadStrategies = async () => {
    try {
      const response = await api.getStrategies();
      if (response.success) {
        setStrategies(response.data || []);
      }
    } catch (error) {
      console.error('Failed to load strategies:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleStatusChange = async (id: number, status: string) => {
    try {
      await api.updateStrategyStatus(id, status);
      await loadStrategies();
    } catch (error) {
      console.error('Failed to update strategy status:', error);
      alert('Failed to update strategy status');
    }
  };

  const handleViewBacktest = async (id: number) => {
    const strategy = strategies.find((s) => s.id === id);
    if (!strategy) return;

    setSelectedStrategy(strategy);
    setShowBacktestModal(true);

    // Load backtest results
    try {
      const response = await api.getBacktestResults(id);
      if (response.success) {
        setBacktestResults(response.data || []);
      }
    } catch (error) {
      console.error('Failed to load backtest results:', error);
      setBacktestResults([]);
    }
  };

  const handleRunBacktest = async (params: any) => {
    setRunningBacktest(true);
    try {
      const response = await api.runBacktest(params);
      if (response.success) {
        alert('Backtest completed successfully!');
        // Reload results
        const resultsResponse = await api.getBacktestResults(params.strategy_id);
        if (resultsResponse.success) {
          setBacktestResults(resultsResponse.data || []);
        }
      }
    } catch (error: any) {
      console.error('Failed to run backtest:', error);
      alert(error.response?.data?.error || 'Failed to run backtest');
    } finally {
      setRunningBacktest(false);
    }
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

  return (
    <div className="min-h-screen bg-gray-50">
      <NavBar />

      <div className="max-w-7xl mx-auto px-4 py-6">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-2xl font-bold text-gray-900">Strategies</h1>
          <button
            onClick={() => router.push('/chat')}
            className="btn btn-primary flex items-center gap-2"
          >
            <Plus className="w-4 h-4" />
            Create Strategy
          </button>
        </div>

        {/* Strategies List */}
        {strategies.length === 0 ? (
          <div className="card text-center py-12">
            <TrendingUp className="w-16 h-16 mx-auto mb-4 text-gray-300" />
            <h2 className="text-xl font-semibold text-gray-900 mb-2">
              No strategies yet
            </h2>
            <p className="text-gray-600 mb-4">
              Upload a Pine Script or create a strategy via AI chat
            </p>
            <button
              onClick={() => router.push('/chat')}
              className="btn btn-primary mx-auto"
            >
              Get Started
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {strategies.map((strategy) => (
              <StrategyCard
                key={strategy.id}
                strategy={strategy}
                onStatusChange={handleStatusChange}
                onViewBacktest={handleViewBacktest}
              />
            ))}
          </div>
        )}

        {/* Backtest Modal */}
        {showBacktestModal && selectedStrategy && (
          <div>
            <BacktestModal
              strategyId={selectedStrategy.id}
              strategyName={selectedStrategy.name}
              onClose={() => setShowBacktestModal(false)}
              onRunBacktest={handleRunBacktest}
            />

            {/* Show results below modal */}
            <div className="fixed inset-x-0 bottom-0 bg-white border-t border-gray-200 p-4 max-h-[50vh] overflow-y-auto">
              <h3 className="font-semibold mb-4">Backtest Results</h3>
              {runningBacktest ? (
                <div className="text-center py-8">
                  <div className="spinner border-primary-600 mx-auto"></div>
                  <p className="text-gray-600 mt-4">Running backtest...</p>
                </div>
              ) : (
                <BacktestResults results={backtestResults} />
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
