import React, { useEffect, useState } from 'react';
import { useRouter } from './router';
import { Layout } from './components/Layout';
import { VerifyModal } from './components/VerifyModal';
import { TransactionDrawer } from './components/TransactionDrawer';
import { DashboardPage } from './pages/DashboardPage';
import { PlaygroundPage } from './pages/PlaygroundPage';
import { TransactionsPage } from './pages/TransactionsPage';
import { ApiKeysPage } from './pages/ApiKeysPage';
import { OrdersPage } from './pages/OrdersPage';
import { JobsPage } from './pages/JobsPage';
import { AuditLogsPage } from './pages/AuditLogsPage';
import { DatabasePage } from './pages/DatabasePage';
import { DocsPage } from './pages/DocsPage';
import { SettingsPage } from './pages/SettingsPage';
import { PublicLandingPage } from './pages/PublicLandingPage';
import { PublicHealthPage } from './pages/PublicHealthPage';
import { BrandLogo } from './components/BrandLogo';
import type { User, Payment } from './types';
import { api } from './api/client';

const VALID_TABS = [
  'dashboard',
  'transactions',
  'orders',
  'jobs',
  'api-keys',
  'playground',
  'docs',
  'settings',
  'database',
  'audit',
];

export const App: React.FC = () => {
  const { path, navigate } = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [isVerifyModalOpen, setIsVerifyModalOpen] = useState(false);
  const [selectedPayment, setSelectedPayment] = useState<Payment | null>(null);

  const cleanPath = (path.startsWith('/') ? path.slice(1) : path).split('?')[0];

  useEffect(() => {
    // Check existing auth token
    if (api.getToken()) {
      api.getMe()
        .then((u) => {
          setUser(u);
          if (cleanPath === '' || cleanPath === 'login' || cleanPath === 'register') {
            navigate('/dashboard', { replace: true });
          }
        })
        .catch(() => {
          api.setToken(null);
          setUser(null);
        })
        .finally(() => setLoading(false));
    } else {
      setLoading(false);
    }

    const handleExpired = () => {
      setUser(null);
      navigate('/');
    };
    window.addEventListener('paygate_auth_expired', handleExpired);

    // Global keyboard shortcut for Quick Verify (Cmd+K / Ctrl+K) inside console
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setIsVerifyModalOpen((prev) => !prev);
      }
    };
    window.addEventListener('keydown', handleKeyDown);

    return () => {
      window.removeEventListener('paygate_auth_expired', handleExpired);
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, []);

  const handleLogout = () => {
    api.setToken(null);
    setUser(null);
    navigate('/');
  };

  const handleLoginSuccess = (loggedInUser: User) => {
    setUser(loggedInUser);
    if (!VALID_TABS.includes(cleanPath)) {
      navigate('/dashboard');
    }
  };

  if (loading) {
    return (
      <div className="min-h-screen bg-[#0c0f0d] flex flex-col items-center justify-center text-slate-400 text-xs font-mono gap-4">
        <BrandLogo size={56} />
        <div className="tracking-wide text-slate-400">Loading authentic-et...</div>
      </div>
    );
  }

  // If user is not authenticated, support public landing, public docs, and public health pages
  if (!user) {
    if (cleanPath === 'docs') {
      return (
        <DocsPage
          isPublic={true}
          onBack={() => navigate('/')}
          onNavigateToHealth={() => navigate('/health')}
          onOpenSignIn={() => navigate('/')}
        />
      );
    }

    if (cleanPath === 'health') {
      return (
        <PublicHealthPage
          onBack={() => navigate('/')}
          onNavigateToDocs={() => navigate('/docs')}
          onOpenSignIn={() => navigate('/')}
        />
      );
    }

    return (
      <PublicLandingPage
        onLoginSuccess={handleLoginSuccess}
        onOpenDocs={() => navigate('/docs')}
        onOpenHealth={() => navigate('/health')}
      />
    );
  }

  // Authenticated route: if user explicitly requested public health view
  if (cleanPath === 'health') {
    return (
      <PublicHealthPage
        onBack={() => navigate('/dashboard')}
        onNavigateToDocs={() => navigate('/docs')}
        onOpenSignIn={() => navigate('/dashboard')}
      />
    );
  }

  const currentTab = VALID_TABS.includes(cleanPath) ? cleanPath : 'dashboard';

  // Authenticated Console
  return (
    <Layout
      currentTab={currentTab}
      onSelectTab={(tab) => navigate(`/${tab}`)}
      user={user}
      onLogout={handleLogout}
      onOpenQuickVerify={() => setIsVerifyModalOpen(true)}
    >
      {currentTab === 'dashboard' && (
        <DashboardPage
          onNavigate={(tab) => navigate(`/${tab}`)}
          onSelectPayment={(p) => setSelectedPayment(p)}
          onOpenVerify={() => setIsVerifyModalOpen(true)}
        />
      )}
      {currentTab === 'playground' && <PlaygroundPage />}
      {currentTab === 'transactions' && (
        <TransactionsPage
          user={user}
          onSelectPayment={(p) => setSelectedPayment(p)}
          onOpenVerify={() => setIsVerifyModalOpen(true)}
        />
      )}
      {currentTab === 'api-keys' && <ApiKeysPage />}
      {currentTab === 'orders' && <OrdersPage />}
      {currentTab === 'jobs' && <JobsPage />}
      {currentTab === 'audit' && <AuditLogsPage />}
      {currentTab === 'database' && <DatabasePage />}
      {currentTab === 'settings' && <SettingsPage />}
      {currentTab === 'docs' && <DocsPage />}

      {/* Global Quick Verify Modal (Cmd+K) */}
      <VerifyModal
        isOpen={isVerifyModalOpen}
        onClose={() => setIsVerifyModalOpen(false)}
        onVerified={() => {
          // Trigger refresh if needed
        }}
      />

      {/* Global Transaction Details Drawer */}
      <TransactionDrawer
        payment={selectedPayment}
        onClose={() => setSelectedPayment(null)}
      />
    </Layout>
  );
};

