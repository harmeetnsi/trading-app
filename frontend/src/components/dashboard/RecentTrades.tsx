
import { format } from 'date-fns';

interface Trade {
  id: number;
  symbol: string;
  action: string;
  quantity: number;
  price: number;
  created_at: string;
}

interface RecentTradesProps {
  trades: Trade[];
}

export default function RecentTrades({ trades }: RecentTradesProps) {
  if (trades.length === 0) {
    return (
      <div className="card">
        <h2 className="text-lg font-semibold mb-4">Recent Trades</h2>
        <p className="text-center text-gray-500 py-8">No recent trades</p>
      </div>
    );
  }

  return (
    <div className="card">
      <h2 className="text-lg font-semibold mb-4">Recent Trades</h2>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-gray-600 border-b border-gray-200">
              <th className="pb-2">Symbol</th>
              <th className="pb-2">Action</th>
              <th className="pb-2">Qty</th>
              <th className="pb-2">Price</th>
              <th className="pb-2">Time</th>
            </tr>
          </thead>
          <tbody>
            {trades.slice(0, 10).map((trade) => (
              <tr key={trade.id} className="border-b border-gray-100">
                <td className="py-2 font-medium">{trade.symbol}</td>
                <td className="py-2">
                  <span
                    className={`px-2 py-1 rounded text-xs font-medium ${
                      trade.action === 'BUY'
                        ? 'bg-green-100 text-green-700'
                        : 'bg-red-100 text-red-700'
                    }`}
                  >
                    {trade.action}
                  </span>
                </td>
                <td className="py-2">{trade.quantity}</td>
                <td className="py-2">₹{trade.price.toFixed(2)}</td>
                <td className="py-2 text-gray-600">
                  {format(new Date(trade.created_at), 'HH:mm')}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
