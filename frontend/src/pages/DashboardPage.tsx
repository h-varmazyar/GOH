import { useEffect } from 'react';
import {
  TrendingUp,
  TrendingDown,
  Activity,
  BarChart3,
  Target,
  Zap,
  AlertTriangle,
  Award,
} from 'lucide-react';
import { Card } from '../components/ui/Card';
import { Badge } from '../components/ui/Badge';
import { useStrategyStore } from '../store/strategyStore';
import { mockDashboardStats } from '../data/mock';

export function DashboardPage() {
  const { strategies, fetchStrategies } = useStrategyStore();

  useEffect(() => {
    fetchStrategies();
  }, [fetchStrategies]);

  const stats = mockDashboardStats;

  const statCards = [
    {
      label: 'سود کل',
      value: `$${stats.totalPnl.toLocaleString()}`,
      icon: TrendingUp,
      color: 'text-success-500',
      bgColor: 'bg-success-400/10',
    },
    {
      label: 'سود امروز',
      value: `$${stats.dailyPnl.toLocaleString()}`,
      icon: stats.dailyPnl >= 0 ? TrendingUp : TrendingDown,
      color: stats.dailyPnl >= 0 ? 'text-success-500' : 'text-danger-500',
      bgColor: stats.dailyPnl >= 0 ? 'bg-success-400/10' : 'bg-danger-400/10',
    },
    {
      label: 'نرخ برد',
      value: `${stats.winRate}%`,
      icon: Target,
      color: 'text-primary-500',
      bgColor: 'bg-primary-400/10',
    },
    {
      label: 'معاملات کل',
      value: stats.totalTrades.toLocaleString(),
      icon: BarChart3,
      color: 'text-primary-500',
      bgColor: 'bg-primary-400/10',
    },
    {
      label: 'استراتژی فعال',
      value: stats.activeStrategies.toString(),
      icon: Zap,
      color: 'text-warning-500',
      bgColor: 'bg-warning-400/10',
    },
    {
      label: 'پوزیشن باز',
      value: stats.openPositions.toString(),
      icon: Activity,
      color: 'text-primary-500',
      bgColor: 'bg-primary-400/10',
    },
    {
      label: 'حداکثر افت',
      value: `${stats.maxDrawdown}%`,
      icon: AlertTriangle,
      color: 'text-danger-500',
      bgColor: 'bg-danger-400/10',
    },
    {
      label: 'شارپ ریشو',
      value: stats.sharpeRatio.toFixed(2),
      icon: Award,
      color: 'text-success-500',
      bgColor: 'bg-success-400/10',
    },
  ];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h1 className="text-2xl font-bold text-surface-900 dark:text-white">داشبورد</h1>
        <p className="text-sm text-surface-500 dark:text-surface-400 mt-1">نمای کلی عملکرد سیستم معاملاتی</p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-3 sm:gap-4">
        {statCards.map((stat) => (
          <Card key={stat.label} padding="sm" className="flex items-center gap-3">
            <div className={`p-2 rounded-lg ${stat.bgColor}`}>
              <stat.icon className={`w-5 h-5 ${stat.color}`} />
            </div>
            <div>
              <p className="text-xs text-surface-500 dark:text-surface-400">{stat.label}</p>
              <p className="text-lg font-bold text-surface-900 dark:text-white">{stat.value}</p>
            </div>
          </Card>
        ))}
      </div>

      {/* Active Strategies */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-surface-900 dark:text-white">استراتژی‌های فعال</h2>
          <Badge variant="primary">{strategies.filter((s) => s.enabled).length} فعال</Badge>
        </div>

        {strategies.length === 0 ? (
          <p className="text-sm text-surface-500 dark:text-surface-400 text-center py-8">
            هنوز استراتژی‌ای ایجاد نشده است
          </p>
        ) : (
          <div className="space-y-3">
            {strategies.map((strategy) => (
              <div
                key={strategy.id}
                className="flex items-center justify-between p-3 rounded-lg bg-surface-50 dark:bg-surface-800/50 border border-surface-100 dark:border-surface-700"
              >
                <div className="flex items-center gap-3">
                  <div className={`w-2 h-2 rounded-full ${strategy.enabled ? 'bg-success-500' : 'bg-surface-400'}`} />
                  <div>
                    <p className="text-sm font-medium text-surface-900 dark:text-white">{strategy.name}</p>
                    <p className="text-xs text-surface-500 dark:text-surface-400">{strategy.provider}</p>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {strategy.tags.slice(0, 2).map((tag) => (
                    <Badge key={tag} size="sm">{tag}</Badge>
                  ))}
                  <Badge variant={strategy.enabled ? 'success' : 'default'} size="sm">
                    {strategy.enabled ? 'فعال' : 'غیرفعال'}
                  </Badge>
                </div>
              </div>
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}
