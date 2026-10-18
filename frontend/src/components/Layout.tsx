import React, { useEffect, useState } from 'react';
import {
  ShieldCheck,
  CreditCard,
  Key,
  Layers,
  Database,
  Terminal,
  FileText,
  LogOut,
  User as UserIcon,
  Activity,
  History,
  Server,
  AlertTriangle,
  Search,
  Sparkles,
  Menu,
  X,
  ChevronRight,
  Zap,
  Sliders,
} from 'lucide-react';
import type { User, SystemReady, DBStatus } from '../types';
import { api } from '../api/client';
import { BrandLogo } from './BrandLogo';
import { Link } from '../router';


interface LayoutProps {
  currentTab: string;
  onSelectTab: (tab: string) => void;
  user: User | null;
  onLogout: () => void;
  onOpenQuickVerify: () => void;
  children: React.ReactNode;
}

export const Layout: React.FC<LayoutProps> = ({
  currentTab,
  onSelectTab,
  user,
  onLogout,
  onOpenQuickVerify,
  children,
}) => {
  const [ready, setReady] = useState<SystemReady | null>(null);
  const [dbStatus, setDbStatus] = useState<DBStatus | null>(null);
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  useEffect(() => {
    api.getSystemReady()
      .then(setReady)
      .catch(() => {});
    api.getDatabaseStatus()
      .then((res) => setDbStatus(res.database || (res as any)))
      .catch(() => {});
  }, [currentTab]);

  const navGroups = [
    {
      title: 'Platform',
      items: [
        { id: 'dashboard', label: 'Overview', icon: Activity },
        { id: 'transactions', label: 'Transactions', icon: CreditCard },
        { id: 'orders', label: 'Orders', icon: Layers },
        { id: 'jobs', label: 'Outbox', icon: Server },
      ],
    },
    {
      title: 'Developers',
      items: [
        { id: 'api-keys', label: 'API Keys', icon: Key },
        { id: 'playground', label: 'Playground', icon: Terminal },
        { id: 'docs', label: 'Documentation', icon: FileText },
      ],
    },
    {
      title: 'System',
      items: [
        { id: 'settings', label: 'Settings', icon: Sliders },
        { id: 'database', label: 'Database', icon: Database },
        { id: 'audit', label: 'Audit Logs', icon: History },
      ],
    },
  ];

  const getPageTitle = (tab: string) => {
    switch (tab) {
      case 'dashboard':
        return 'Overview';
      case 'transactions':
        return 'Transactions';
      case 'orders':
        return 'Orders';
      case 'jobs':
        return 'Outbox';
      case 'api-keys':
        return 'API Keys';
      case 'playground':
        return 'Playground';
      case 'docs':
        return 'Documentation';
      case 'settings':
        return 'Settings';
      case 'database':
        return 'Database';
      case 'audit':
        return 'Audit Logs';
      default:
        return 'Console';
    }
  };

  const pageTitle = getPageTitle(currentTab);

  return (
    <div className="min-h-screen flex bg-[#0c0f0d] text-slate-100 antialiased font-sans selection:bg-[#00D26A] selection:text-black">
      {/* Desktop Sidebar (Flat, No harsh borders) */}
      <aside className="hidden lg:flex lg:flex-col w-60 border-r border-white/[0.06] bg-[#0c0f0d] sticky top-0 h-screen shrink-0 z-30">
        {/* Brand */}
        <div className="h-16 px-5 border-b border-white/[0.06] flex items-center justify-between">
          <Link to="/dashboard" className="flex items-center gap-2">
            <BrandLogo size={32} showText={true} />
          </Link>
        </div>

        {/* Navigation Items */}
        <div className="flex-1 overflow-y-auto px-3 py-5 space-y-5 scrollbar-none">
          {navGroups.map((group) => (
            <div key={group.title} className="space-y-1">
              <div className="px-2 text-[10px] font-semibold uppercase tracking-wider text-slate-500">
                {group.title}
              </div>
              <div className="space-y-0.5">
                {group.items.map((item) => {
                  const Icon = item.icon;
                  const isActive = currentTab === item.id;
                  return (
                    <Link
                      key={item.id}
                      to={`/${item.id}`}
                      className={`w-full flex items-center justify-between px-2.5 py-1.5 rounded-lg text-xs font-medium transition ${
                        isActive
                          ? 'bg-white/[0.08] text-white'
                          : 'text-slate-400 hover:text-slate-200 hover:bg-white/[0.03]'
                      }`}
                    >
                      <div className="flex items-center gap-2.5 truncate">
                        <Icon className={`w-3.5 h-3.5 ${isActive ? 'text-[#00D26A]' : 'text-slate-500'}`} />
                        <span className="truncate">{item.label}</span>
                      </div>
                      {isActive && <span className="w-1.5 h-1.5 rounded-full bg-[#00D26A]" />}
                    </Link>
                  );
                })}
              </div>
            </div>
          ))}
        </div>

        {/* Sidebar Footer */}
        <div className="p-3 border-t border-white/[0.06] bg-black/20 space-y-2">
          {/* Engine Status Card */}
          <Link
            to="/database"
            className="p-2 rounded-lg bg-white/[0.03] hover:bg-white/[0.06] cursor-pointer transition flex items-center justify-between block"
          >
            <div className="flex items-center gap-2">
              <span className="h-1.5 w-1.5 rounded-full bg-[#00D26A]" />
              <div>
                <div className="text-[10px] text-slate-500 font-medium">Engine</div>
                <div className="text-xs font-medium text-slate-200 capitalize">
                  {dbStatus?.driver === 'sqlite' ? 'SQLite (WAL)' : dbStatus?.driver === 'postgres' ? 'PostgreSQL' : dbStatus?.driver || 'SQLite'}
                </div>
              </div>
            </div>
            <Database className="w-3.5 h-3.5 text-slate-500" />
          </Link>

          {/* User Account / Sign Out */}
          {user && (
            <div className="flex items-center justify-between p-2 rounded-lg bg-white/[0.02]">
              <div className="flex items-center gap-2 truncate">
                <div className="h-6 w-6 rounded-full bg-[#00D26A]/20 text-[#00D26A] flex items-center justify-center text-xs font-bold shrink-0">
                  {user.name.charAt(0).toUpperCase()}
                </div>
                <div className="truncate text-left">
                  <div className="text-xs font-medium text-slate-200 truncate">{user.name}</div>
                  <div className="text-[10px] text-slate-500 font-mono uppercase">{user.role}</div>
                </div>
              </div>
              <button
                onClick={onLogout}
                title="Sign Out to Portal"
                className="p-1.5 rounded text-slate-500 hover:text-rose-400 transition shrink-0"
              >
                <LogOut className="w-3.5 h-3.5" />
              </button>
            </div>
          )}
        </div>
      </aside>

      {/* Main App Container */}
      <div className="flex-1 flex flex-col min-w-0">
        {/* Top Header */}
        <header className="h-16 border-b border-white/[0.06] bg-[#0c0f0d]/80 backdrop-blur-md sticky top-0 z-20 px-4 sm:px-6 flex items-center justify-between">
          <div className="flex items-center gap-3">
            {/* Mobile Hamburger */}
            <button
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              className="lg:hidden p-1.5 rounded text-slate-400 hover:text-white hover:bg-white/[0.05] transition"
            >
              {mobileMenuOpen ? <X className="w-5 h-5" /> : <Menu className="w-5 h-5" />}
            </button>

            <div>
              <h1 className="text-sm font-semibold text-white tracking-tight">{pageTitle}</h1>
            </div>
          </div>

          {/* Header Action Items */}
          <div className="flex items-center gap-2.5">
            {/* Quick Verify Modal Trigger (Flat) */}
            <button
              onClick={onOpenQuickVerify}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-[#00D26A] hover:bg-[#25F28A] text-black font-semibold text-xs transition"
            >
              <Zap className="w-3.5 h-3.5 fill-black" />
              <span>Verify</span>
              <span className="hidden sm:inline-block px-1 py-0.2 rounded bg-black/20 text-[10px] font-mono text-black font-bold">
                ⌘K
              </span>
            </button>
          </div>
        </header>

        {/* Mobile Slide-down Menu */}
        {mobileMenuOpen && (
          <div className="lg:hidden border-b border-white/[0.06] bg-[#0c0f0d] px-4 py-4 space-y-4">
            {navGroups.map((group) => (
              <div key={group.title} className="space-y-1">
                <div className="text-[10px] font-semibold uppercase tracking-wider text-slate-500 px-2">{group.title}</div>
                {group.items.map((item) => (
                  <Link
                    key={item.id}
                    to={`/${item.id}`}
                    onClick={() => setMobileMenuOpen(false)}
                    className={`w-full flex items-center gap-2.5 px-3 py-2 rounded-lg text-xs font-medium ${
                      currentTab === item.id ? 'bg-white/[0.08] text-white' : 'text-slate-400 hover:text-white'
                    }`}
                  >
                    <item.icon className="w-4 h-4" />
                    <span>{item.label}</span>
                  </Link>
                ))}
              </div>
            ))}
          </div>
        )}

        {/* Page Content Viewport */}
        <main className="flex-1 p-4 sm:p-6 lg:p-8 max-w-7xl w-full mx-auto">
          {children}
        </main>

        {/* Minimalist Clean Footer */}
        <footer className="border-t border-white/[0.06] px-6 py-4 flex flex-col sm:flex-row items-center justify-between text-xs text-slate-500 gap-2">
          <div>authentic-et</div>
          <div className="flex items-center gap-4">
            <span>Open Source</span>
          </div>
        </footer>
      </div>
    </div>
  );
};
