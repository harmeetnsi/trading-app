
'use client';

import { useState } from 'react';
import { X, TrendingUp } from 'lucide-react';

interface BacktestModalProps {
  strategyId: number;
  strategyName: string;
  onClose: () => void;
  onRunBacktest: (params: any) => void;
}

export default function BacktestModal({
  strategyId,
  strategyName,
  onClose,
  onRunBacktest,
}: BacktestModalProps) {
  const [startDate, setStartDate] = useState('2024-01-01');
  const [endDate, setEndDate] = useState('2024-12-31');
  const [initialCapital, setInitialCapital] = useState('100000');
  const [symbol, setSymbol] = useState('SBIN');
  const [exchange, setExchange] = useState('NSE');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onRunBacktest({
      strategy_id: strategyId,
      start_date: startDate,
      end_date: endDate,
      initial_capital: parseFloat(initialCapital),
      symbol,
      exchange,
    });
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div className="bg-white rounded-lg max-w-md w-full p-6">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <TrendingUp className="w-5 h-5 text-primary-600" />
            <h2 className="text-xl font-semibold">Run Backtest</h2>
          </div>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        <p className="text-sm text-gray-600 mb-4">
          Strategy: <span className="font-medium">{strategyName}</span>
        </p>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Start Date
              </label>
              <input
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                className="input"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                End Date
              </label>
              <input
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                className="input"
                required
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Initial Capital (₹)
            </label>
            <input
              type="number"
              value={initialCapital}
              onChange={(e) => setInitialCapital(e.target.value)}
              className="input"
              required
              min="1000"
              step="1000"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Symbol
              </label>
              <input
                type="text"
                value={symbol}
                onChange={(e) => setSymbol(e.target.value.toUpperCase())}
                className="input"
                required
                placeholder="SBIN"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Exchange
              </label>
              <select
                value={exchange}
                onChange={(e) => setExchange(e.target.value)}
                className="input"
                required
              >
                <option value="NSE">NSE</option>
                <option value="BSE">BSE</option>
                <option value="NFO">NFO</option>
              </select>
            </div>
          </div>

          <div className="flex gap-2 mt-6">
            <button type="button" onClick={onClose} className="btn btn-secondary flex-1">
              Cancel
            </button>
            <button type="submit" className="btn btn-primary flex-1">
              Run Backtest
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
