
interface BacktestResult {
  id: number;
  total_return: number;
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  max_drawdown: number;
  sharpe_ratio: number;
  created_at: string;
}

interface BacktestResultsProps {
  results: BacktestResult[];
}

export default function BacktestResults({ results }: BacktestResultsProps) {
  if (results.length === 0) {
    return (
      <div className="text-center text-gray-500 py-8">
        No backtest results yet. Run a backtest to see results.
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {results.map((result) => {
        const winRate =
          result.total_trades > 0
            ? (result.winning_trades / result.total_trades) * 100
            : 0;

        return (
          <div key={result.id} className="card">
            <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
              <div>
                <p className="text-sm text-gray-600">Total Return</p>
                <p
                  className={`text-xl font-bold ${
                    result.total_return >= 0 ? 'text-green-600' : 'text-red-600'
                  }`}
                >
                  {result.total_return >= 0 ? '+' : ''}
                  {result.total_return.toFixed(2)}%
                </p>
              </div>

              <div>
                <p className="text-sm text-gray-600">Total Trades</p>
                <p className="text-xl font-bold text-gray-900">
                  {result.total_trades}
                </p>
              </div>

              <div>
                <p className="text-sm text-gray-600">Win Rate</p>
                <p className="text-xl font-bold text-gray-900">
                  {winRate.toFixed(1)}%
                </p>
              </div>

              <div>
                <p className="text-sm text-gray-600">Max Drawdown</p>
                <p className="text-xl font-bold text-red-600">
                  {result.max_drawdown.toFixed(2)}%
                </p>
              </div>

              <div>
                <p className="text-sm text-gray-600">Sharpe Ratio</p>
                <p className="text-xl font-bold text-gray-900">
                  {result.sharpe_ratio.toFixed(2)}
                </p>
              </div>

              <div>
                <p className="text-sm text-gray-600">W/L</p>
                <p className="text-xl font-bold text-gray-900">
                  {result.winning_trades}/{result.losing_trades}
                </p>
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
