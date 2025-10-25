
import { Play, Pause, TrendingUp } from 'lucide-react';

interface Strategy {
  id: number;
  name: string;
  description: string;
  status: string;
  created_at: string;
}

interface StrategyCardProps {
  strategy: Strategy;
  onStatusChange: (id: number, status: string) => void;
  onViewBacktest: (id: number) => void;
}

export default function StrategyCard({
  strategy,
  onStatusChange,
  onViewBacktest,
}: StrategyCardProps) {
  const isActive = strategy.status === 'active';

  return (
    <div className="card hover:shadow-md transition-shadow">
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1">
          <h3 className="text-lg font-semibold text-gray-900">{strategy.name}</h3>
          <p className="text-sm text-gray-600 mt-1">{strategy.description}</p>
        </div>
        <span
          className={`px-3 py-1 rounded-full text-xs font-medium ${
            isActive
              ? 'bg-green-100 text-green-700'
              : 'bg-gray-100 text-gray-700'
          }`}
        >
          {strategy.status}
        </span>
      </div>

      <div className="flex items-center gap-2 mt-4">
        <button
          onClick={() => onStatusChange(strategy.id, isActive ? 'paused' : 'active')}
          className={`btn flex items-center gap-2 flex-1 ${
            isActive ? 'btn-secondary' : 'btn-success'
          }`}
        >
          {isActive ? (
            <>
              <Pause className="w-4 h-4" />
              Pause
            </>
          ) : (
            <>
              <Play className="w-4 h-4" />
              Activate
            </>
          )}
        </button>
        <button
          onClick={() => onViewBacktest(strategy.id)}
          className="btn btn-secondary flex items-center gap-2 flex-1"
        >
          <TrendingUp className="w-4 h-4" />
          Backtest
        </button>
      </div>
    </div>
  );
}
