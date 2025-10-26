
interface PortfolioCardProps {
  title: string;
  value: string | number;
  change?: number;
  subtitle?: string;
  variant?: 'default' | 'success' | 'error';
}

export default function PortfolioCard({
  title,
  value,
  change,
  subtitle,
  variant = 'default',
}: PortfolioCardProps) {
  const colorClass =
    variant === 'success'
      ? 'text-green-600'
      : variant === 'error'
      ? 'text-red-600'
      : 'text-gray-900';

  return (
    <div className="card">
      <h3 className="text-sm font-medium text-gray-600 mb-1">{title}</h3>
      <div className="flex items-baseline gap-2">
        <p className={`text-2xl font-bold ${colorClass}`}>{value}</p>
        {change !== undefined && (
          <span
            className={`text-sm font-medium ${
              change >= 0 ? 'text-green-600' : 'text-red-600'
            }`}
          >
            {change >= 0 ? '+' : ''}
            {change.toFixed(2)}%
          </span>
        )}
      </div>
      {subtitle && <p className="text-xs text-gray-500 mt-1">{subtitle}</p>}
    </div>
  );
}
