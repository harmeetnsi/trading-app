
interface Position {
  symbol: string;
  quantity: number;
  averageprice: number;
  pnl: number;
}

interface PositionsListProps {
  positions: Position[];
}

export default function PositionsList({ positions }: PositionsListProps) {
  if (positions.length === 0) {
    return (
      <div className="card">
        <h2 className="text-lg font-semibold mb-4">Open Positions</h2>
        <p className="text-center text-gray-500 py-8">No open positions</p>
      </div>
    );
  }

  return (
    <div className="card">
      <h2 className="text-lg font-semibold mb-4">Open Positions</h2>
      <div className="space-y-2">
        {positions.map((position, index) => (
          <div
            key={index}
            className="flex items-center justify-between p-3 bg-gray-50 rounded-lg"
          >
            <div>
              <p className="font-medium text-gray-900">{position.symbol}</p>
              <p className="text-sm text-gray-600">
                {position.quantity} @ ₹{position.averageprice.toFixed(2)}
              </p>
            </div>
            <div className="text-right">
              <p
                className={`font-medium ${
                  position.pnl >= 0 ? 'text-green-600' : 'text-red-600'
                }`}
              >
                {position.pnl >= 0 ? '+' : ''}₹{position.pnl.toFixed(2)}
              </p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
