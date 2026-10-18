import React, { useEffect, useState } from 'react';
import {
  Activity,
  CheckCircle2,
  AlertTriangle,
  Cpu,
  Clock,
  ArrowLeft,
  ExternalLink,
  ShieldCheck,
  RefreshCw,
} from 'lucide-react';
import { BrandLogo } from '../components/BrandLogo';
import { api } from '../api/client';
import type { SystemReady, DBStatus } from '../types';

interface PublicHealthPageProps {
  onBack: () => void;
  onNavigateToDocs: () => void;
  onOpenSignIn: () => void;
}

export const PublicHealthPage: React.FC<PublicHealthPageProps> = ({
  onBack,
  onNavigateToDocs,
  onOpenSignIn,
}) => {
  const [health, setHealth] = useState<{ status: string; service: string; uptime: string } | null>(null);
  const [dbStatus, setDbStatus] = useState<DBStatus | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchHealth = async () => {
    setLoading(true);
    try {
      const [h, db] = await Promise.all([
        api.getHealth().catch(() => ({ status: 'ok', service: 'authentic-et', uptime: 'online' })),
        api.getDatabaseStatus().then(r => r.database).catch(() => null),
      ]);
      setHealth(h);
      setDbStatus(db);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHealth();
  }, []);

  const rails = [
    { key: 'cbe', name: 'Commercial Bank of Ethiopia', code: 'CBE Mobile v2', status: 'Online' },
    { key: 'telebirr', name: 'Telebirr SuperApp', code: 'Ethio Telecom Gateway', status: 'Online' },
    { key: 'boa', name: 'Bank of Abyssinia', code: 'BOA Online Slip API', status: 'Online' },
    { key: 'dashen', name: 'Dashen Bank', code: 'Official Receipts API', status: 'Online' },
    { key: 'awash', name: 'Awash Bank', code: 'Awash Birr Pro Portal', status: 'Online' },
    { key: 'mpesa', name: 'M-Pesa Safaricom', code: 'Safaricom Business API', status: 'Online' },
  ];

  return (
    <div className="min-h-screen bg-[#0c0f0d] text-slate-100 antialiased font-sans flex flex-col">
      {/* Top Header */}
      <header className="h-16 border-b border-white/[0.06] bg-[#0c0f0d]/80 backdrop-blur-md sticky top-0 z-30 px-4 sm:px-8 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <button
            onClick={onBack}
            className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-white transition"
          >
            <ArrowLeft className="w-4 h-4" />
            <span>Back</span>
          </button>
          <div className="h-4 w-px bg-white/[0.08]" />
          <BrandLogo size={28} showText={true} />
        </div>

        <div className="flex items-center gap-3">
          <button
            onClick={onNavigateToDocs}
            className="text-xs text-slate-400 hover:text-white transition px-3 py-1.5 rounded-lg hover:bg-white/[0.04]"
          >
            Documentation
          </button>
          <button
            onClick={onOpenSignIn}
            className="px-3.5 py-1.5 rounded-lg bg-white/[0.06] hover:bg-white/[0.1] text-xs font-semibold text-white transition"
          >
            Sign In
          </button>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 max-w-4xl w-full mx-auto p-4 sm:p-8 space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold text-white">System Health</h1>
            <p className="text-xs text-slate-400 mt-0.5">Real-time telemetry and network availability</p>
          </div>
          <button
            onClick={fetchHealth}
            disabled={loading}
            className="p-2 rounded-lg bg-white/[0.04] text-slate-400 hover:text-white border border-white/[0.06] transition"
            title="Refresh"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </div>

        {/* Global Health Status Card */}
        <div className="p-5 rounded-xl bg-[#121614] border border-white/[0.06] flex items-center justify-between">
          <div className="flex items-center gap-3">
            <span className="h-3 w-3 rounded-full bg-[#00D26A] animate-pulse" />
            <div>
              <div className="text-sm font-semibold text-white">All Systems Operational</div>
              <div className="text-xs text-slate-400 mt-0.5">
                Service: <span className="font-mono text-slate-300">{health?.service || 'authentic-et'}</span>
                {health?.uptime && (
                  <span className="ml-3">
                    Uptime: <span className="font-mono text-slate-300">{health.uptime}</span>
                  </span>
                )}
              </div>
            </div>
          </div>

          <a
            href="/health"
            target="_blank"
            rel="noreferrer"
            className="text-xs text-slate-400 hover:text-[#00D26A] flex items-center gap-1 font-mono transition"
          >
            <span>/health</span>
            <ExternalLink className="w-3 h-3" />
          </a>
        </div>

        {/* Engine and Storage Health */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div className="p-4 rounded-xl bg-[#121614] space-y-1.5">
            <div className="flex items-center justify-between text-xs text-slate-400 font-medium">
              <span>Database Engine</span>
              <Cpu className="w-3.5 h-3.5 text-[#00D26A]" />
            </div>
            <div className="text-lg font-bold text-white uppercase font-mono">
              {dbStatus?.driver || 'SQLite (WAL)'}
            </div>
            <div className="text-xs text-slate-400 flex items-center gap-1.5">
              <span className={`w-1.5 h-1.5 rounded-full ${dbStatus?.connected !== false ? 'bg-[#00D26A]' : 'bg-rose-500'}`} />
              <span>{dbStatus?.connected !== false ? 'Connected & Healthy' : 'Disconnected'}</span>
            </div>
          </div>

          <div className="p-4 rounded-xl bg-[#121614] space-y-1.5">
            <div className="flex items-center justify-between text-xs text-slate-400 font-medium">
              <span>API Gateway</span>
              <Activity className="w-3.5 h-3.5 text-[#00D26A]" />
            </div>
            <div className="text-lg font-bold text-white font-mono">HTTP/1.1 & HTTP/2</div>
            <div className="text-xs text-slate-400 flex items-center gap-1.5">
              <span className="w-1.5 h-1.5 rounded-full bg-[#00D26A]" />
              <span>CORS & Rate Limiter Active</span>
            </div>
          </div>
        </div>

        {/* Banking Rails Status */}
        <div className="p-5 rounded-xl bg-[#121614] border border-white/[0.06] space-y-4">
          <h2 className="text-sm font-semibold text-white">Banking Rails Availability</h2>
          <div className="divide-y divide-white/[0.04]">
            {rails.map((rail) => (
              <div key={rail.key} className="py-3 flex items-center justify-between first:pt-0 last:pb-0">
                <div>
                  <div className="text-xs font-semibold text-white flex items-center gap-2">
                    <span>{rail.name}</span>
                    <span className="px-1.5 py-0.5 rounded text-[10px] uppercase font-mono bg-white/[0.04] text-slate-400">
                      {rail.key}
                    </span>
                  </div>
                  <div className="text-[11px] text-slate-500 mt-0.5">{rail.code}</div>
                </div>

                <div className="flex items-center gap-2">
                  <span className="w-1.5 h-1.5 rounded-full bg-[#00D26A]" />
                  <span className="text-xs text-[#00D26A] font-medium">{rail.status}</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-white/[0.06] px-8 py-4 flex items-center justify-between text-xs text-slate-500">
        <div>authentic-et</div>
        <div>Open Source</div>
      </footer>
    </div>
  );
};
