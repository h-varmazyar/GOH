import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { useEffect } from 'react';
import { useAuthStore } from './store/authStore';
import { useThemeStore } from './store/themeStore';
import { AppLayout } from './components/layout/AppLayout';
import { LoginPage } from './pages/LoginPage';
import { RegisterPage } from './pages/RegisterPage';
import { ForgotPasswordPage } from './pages/ForgotPasswordPage';
import { VerifyPhonePage } from './pages/VerifyPhonePage';
import { DashboardPage } from './pages/DashboardPage';
import { ProfilePage } from './pages/ProfilePage';
import { StrategiesPage } from './pages/StrategiesPage';
import { CreateStrategyPage } from './pages/CreateStrategyPage';
import { StrategyDetailPage } from './pages/StrategyDetailPage';
import { StrategyTemplatesPage } from './pages/StrategyTemplatesPage';
import { TelegramOverview } from './pages/telegram/TelegramOverview';
import { TelegramAddAccount } from './pages/telegram/TelegramAddAccount';
import { TelegramAccountsList } from './pages/telegram/TelegramAccountsList';
import { TelegramAuth } from './pages/telegram/TelegramAuth';
import { TelegramAccountSettings } from './pages/telegram/TelegramAccountSettings';
import { TelegramProxiesList } from './pages/telegram/TelegramProxiesList';
import { AirdropGiftsPage } from './pages/airdrop/AirdropGiftsPage';
import { KeywordRoutesPage } from './pages/airdrop/KeywordRoutesPage';
import { SubscriptionFiltersPage } from './pages/airdrop/SubscriptionFiltersPage';
import { BackendConfigsPage } from './pages/backend/BackendConfigsPage';
import { WorkersPage } from './pages/backend/WorkersPage';
import { TseOverviewPage } from './pages/tse/TseOverviewPage';
import { DailyAnalysisListPage } from './pages/tse/DailyAnalysisListPage';
import { DailyAnalysisDetailPage } from './pages/tse/DailyAnalysisDetailPage';
import { IndicatorsPage } from './pages/tse/IndicatorsPage';
import { MarketsPage } from './pages/tse/MarketsPage';
import { WatchlistsPage } from './pages/tse/WatchlistsPage';
import { MarketCalendarPage } from './pages/tse/MarketCalendarPage';

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, user } = useAuthStore();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  if (user && !user.isVerified) return <Navigate to="/verify-phone" replace />;
  return <>{children}</>;
}

function GuestRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, user } = useAuthStore();
  if (isAuthenticated && user?.isVerified) return <Navigate to="/dashboard" replace />;
  return <>{children}</>;
}

function VerifyRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, user } = useAuthStore();
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  if (user?.isVerified) return <Navigate to="/dashboard" replace />;
  return <>{children}</>;
}

export default function App() {
  const { theme } = useThemeStore();

  useEffect(() => {
    document.documentElement.classList.toggle('dark', theme === 'dark');
  }, [theme]);

  return (
    <BrowserRouter>
      <Routes>
        {/* Auth Routes */}
        <Route path="/login" element={<GuestRoute><LoginPage /></GuestRoute>} />
        <Route path="/register" element={<GuestRoute><RegisterPage /></GuestRoute>} />
        <Route path="/forgot-password" element={<GuestRoute><ForgotPasswordPage /></GuestRoute>} />
        <Route path="/verify-phone" element={<VerifyRoute><VerifyPhonePage /></VerifyRoute>} />

        {/* Protected Routes */}
        <Route path="/" element={<ProtectedRoute><AppLayout /></ProtectedRoute>}>
          <Route index element={<Navigate to="/dashboard" replace />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="profile" element={<ProfilePage />} />
          <Route path="strategies" element={<StrategiesPage />} />
          <Route path="strategies/new" element={<CreateStrategyPage />} />
          <Route path="strategies/templates" element={<StrategyTemplatesPage />} />
          <Route path="strategies/:id" element={<StrategyDetailPage />} />

          {/* Telegram */}
          <Route path="telegram" element={<TelegramOverview />} />
          <Route path="telegram/add" element={<TelegramAddAccount />} />
          <Route path="telegram/accounts" element={<TelegramAccountsList />} />
          <Route path="telegram/auth/:accountId" element={<TelegramAuth />} />
          <Route path="telegram/settings/:accountId" element={<TelegramAccountSettings />} />
          <Route path="telegram/proxies" element={<TelegramProxiesList />} />

          {/* Airdrop */}
          <Route path="airdrop/gifts" element={<AirdropGiftsPage />} />
          <Route path="airdrop/keywords" element={<KeywordRoutesPage />} />
          <Route path="airdrop/subscriptions" element={<SubscriptionFiltersPage />} />

          {/* Tehran Stock Exchange */}
          <Route path="tse" element={<TseOverviewPage />} />
          <Route path="tse/analysis" element={<DailyAnalysisListPage />} />
          <Route path="tse/analysis/:analysisId" element={<DailyAnalysisDetailPage />} />
          <Route path="tse/indicators" element={<IndicatorsPage />} />
          <Route path="tse/markets" element={<MarketsPage />} />
          <Route path="tse/watchlist" element={<WatchlistsPage />} />
          <Route path="tse/calendar" element={<MarketCalendarPage />} />

          {/* Backend management */}
          <Route path="backend/configs" element={<BackendConfigsPage />} />
          <Route path="backend/workers" element={<WorkersPage />} />
        </Route>

        <Route path="*" element={<Navigate to="/login" replace />} />
      </Routes>
    </BrowserRouter>
  );
}
