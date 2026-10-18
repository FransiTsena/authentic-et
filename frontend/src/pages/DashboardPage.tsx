import React, { useEffect, useState } from 'react';
import {
  Activity,
  CheckCircle2,
  XCircle,
  Clock,
  ArrowUpRight,
  ShieldCheck,
  CreditCard,
  Key,
  Terminal,
  Zap,
  ArrowRight,
  Sparkles,
  ExternalLink,
  ChevronRight,
  Copy,
  Check,
} from 'lucide-react';
import type { VerificationStats, Payment } from '../types';
import { api } from '../api/client';

interface DashboardProps {
  onNavigate: (tab: string) => void;
  onSelectPayment: (payment: Payment) => void;
  onOpenVerify: () => void;
}

export const DashboardPage: React.FC<DashboardProps> = ({
  onNavigate,
  onSelectPayment,
  onOpenVerify,
}) => {
  const [stats, setStats] = useState<VerificationStats | null>(null);
  const [recentTx, setRecentTx] = useState<Payment[]>([]);
  const [loading, setLoading] = useState(true);
  const [copiedSnippet, setCopiedSnippet] = useState(false);

  useEffect(() => {
    Promise.all([
      api.getStats().catch(() => null),
      api.listTransactions({ limit: 8 }).catch(() => ({ items: [] })),
    ]).then(([statsData, txData]) => {
      if (statsData) setStats(statsData);
      if (txData && txData.items) setRecentTx(txData.items);
      setLoading(false);
    });
  }, []);

  const curlSnippet = `curl -X POST https://api.yourdomain.com/api/v1/public/payment/verify \\
  -H "Content-Type: application/json" \\
  -d '{
    "provider": "cbe",
    "transactionId": "FT262036S1GH10104",
    "expectedAmount": 1250.00
  }'`;

  const copySnippet = () => {
    navigator.clipboard.writeText(curlSnippet);
    setCopiedSnippet(true);
    setTimeout(() => setCopiedSnippet(false), 2000);
  };

  const getProviderBadge = (key: string) => {
    switch (key.toLowerCase()) {
      case 'cbe':
        return 'text-purple-400 bg-purple-500/10 border-purple-500/20';
      case 'telebirr':
        return 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20';
      case 'boa':
        return 'text-amber-400 bg-amber-500/10 border-amber-500/20';
      case 'dashen':
        return 'text-blue-400 bg-blue-500/10 border-blue-500/20';
      case 'awash':
        return 'text-orange-400 bg-orange-500/10 border-orange-500/20';
      case 'mpesa':
        return 'text-rose-400 bg-rose-500/10 border-rose-500/20';
      default:
        return 'text-slate-400 bg-slate-500/10 border-slate-500/20';
    }
  };

  const bankRails = [
    { key: 'cbe', name: 'Commercial Bank of Ethiopia', code: 'CBE Mobile v2', color: 'purple' },
    { key: 'telebirr', name: 'Telebirr', code: 'Ethio Telecom Gateway', color: 'emerald' },
    { key: 'boa', name: 'Bank of Abyssinia', code: 'BOA Online Slip API', color: 'amber' },
    { key: 'dashen', name: 'Dashen Bank', code: 'Official Receipts API', color: 'blue' },
    { key: 'awash', name: 'Awash Bank', code: 'Awash Birr Pro Portal', color: 'orange' },
    { key: 'mpesa', name: 'M-Pesa Ethiopia', code: 'Safaricom Business API', color: 'rose' },
  ];

  if (loading) {
    return (
      <div className="space-y-6 animate-pulse">
        <div className="h-28 bg-[#121614] border border-white/[0.06] rounded-xl"></div>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-28 bg-[#121614] border border-white/[0.06] rounded-xl"></div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Primary KPI Metrics */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3.5">
        <div className="p-4 rounded-xl bg-[#121614]">
          <div className="flex items-center justify-between text-slate-400 mb-1.5">
            <span className="text-xs font-medium text-slate-400">Total Volume</span>
            <Zap className="w-3.5 h-3.5 text-[#00D26A]" />
          </div>
          <div className="text-xl font-bold text-white tracking-tight font-mono">
            {(stats?.totalVerifiedAmount || 0).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} <span className="text-xs font-sans text-slate-400 font-normal">ETB</span>
          </div>
        </div>

        <div className="p-4 rounded-xl bg-[#121614]">
          <div className="flex items-center justify-between text-slate-400 mb-1.5">
            <span className="text-xs font-medium text-slate-400">Success Rate</span>
            <CheckCircle2 className="w-3.5 h-3.5 text-[#00D26A]" />
          </div>
          <div className="text-xl font-bold text-[#00D26A] tracking-tight font-mono">
            {(stats?.successRatePercent || 0).toFixed(1)}%
          </div>
        </div>

        <div className="p-4 rounded-xl bg-[#121614]">
          <div className="flex items-center justify-between text-slate-400 mb-1.5">
            <span className="text-xs font-medium text-slate-400">Settled Volume</span>
            <ShieldCheck className="w-3.5 h-3.5 text-purple-400" />
          </div>
          <div className="text-xl font-bold text-white tracking-tight font-mono">
            {(stats?.totalClaimedAmount || 0).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })} <span className="text-xs font-sans text-slate-400 font-normal">ETB</span>
          </div>
        </div>

        <div className="p-4 rounded-xl bg-[#121614]">
          <div className="flex items-center justify-between text-slate-400 mb-1.5">
            <span className="text-xs font-medium text-slate-400">Verifications</span>
            <Activity className="w-3.5 h-3.5 text-amber-400" />
          </div>
          <div className="text-xl font-bold text-white tracking-tight font-mono">
            {(stats?.totalVerifications || 0).toLocaleString()}
          </div>
        </div>
      </div>

      {/* Multi-Rail Banking Infrastructure Matrix */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-white tracking-tight">Banking Rails</h2>
          <button
            onClick={() => onNavigate('playground')}
            className="text-xs text-[#00D26A] hover:text-[#25F28A] font-medium flex items-center gap-1 transition"
          >
            <span>Playground</span>
            <ChevronRight className="w-3.5 h-3.5" />
          </button>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {bankRails.map((rail) => {
            const count = stats?.providers?.[rail.key] || 0;
            return (
              <div
                key={rail.key}
                onClick={onOpenVerify}
                className="p-3.5 rounded-xl bg-[#121614] hover:bg-[#151a17] transition cursor-pointer group flex flex-col justify-between"
              >
                <div>
                  <div className="text-xs font-semibold text-white mt-2 group-hover:text-[#00D26A] transition">
                    {rail.name}
                    <span className="h-1.5 w-1.5 rounded-full bg-[#00D26A]" />
                  </div>
                  <div className="text-[11px] text-slate-500 mt-0.5">{rail.code}</div>
                </div>

                <div className="mt-3 pt-2.5 border-t border-white/[0.06] flex items-center justify-between text-xs">
                  <span className="text-[11px] text-slate-500">Verified</span>
                  <span className="font-mono font-medium text-slate-200">{count.toLocaleString()}</span>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Live Recent Transactions Feed */}
      <div className="p-5 rounded-xl bg-[#121614] border border-white/[0.06] space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-white tracking-tight">Recent Transactions</h2>
          <button
            onClick={() => onNavigate('transactions')}
            className="flex items-center gap-1 text-xs text-slate-400 hover:text-white font-medium transition"
          >
            <span>View all</span>
            <ArrowUpRight className="w-3.5 h-3.5" />
          </button>
        </div>

        {recentTx.length === 0 ? (
          <div className="text-center py-10 rounded-lg bg-black/20 space-y-2">
            <CreditCard className="w-6 h-6 text-slate-600 mx-auto" />
            <div className="text-xs text-slate-400 font-medium">No verified transactions yet</div>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead>
                <tr className="border-b border-white/[0.06] text-slate-500 font-medium uppercase tracking-wider text-[10px]">
                  <th className="py-2.5 px-3">Rail</th>
                  <th className="py-2.5 px-3">Transaction ID</th>
                  <th className="py-2.5 px-3">Amount</th>
                  <th className="py-2.5 px-3">Payer / Receiver</th>
                  <th className="py-2.5 px-3">Status</th>
                  <th className="py-2.5 px-3">Claim</th>
                  <th className="py-2.5 px-3">Timestamp</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/[0.04]">
                {recentTx.map((tx) => (
                  <tr
                    key={tx.id}
                    onClick={() => onSelectPayment(tx)}
                    className="hover:bg-white/[0.02] transition cursor-pointer group"
                  >
                    
                    <td className="py-2.5 px-3 font-mono text-slate-200 group-hover:text-[#00D26A] transition font-medium">
                      {tx.transactionId}
                    </td>
                    <td className="py-2.5 px-3 font-bold text-white font-mono">
                      {tx.amount.toLocaleString('en-US', { minimumFractionDigits: 2 })} {tx.currency || 'ETB'}
                    </td>
                    <td className="py-2.5 px-3 text-slate-400 truncate max-w-[180px]">
                      {tx.payerIdentifier || '—'}
                    </td>
                    <td className="py-2.5 px-3">
                      <span
                        className={`inline-flex items-center px-2 py-0.5 rounded text-[10px] font-semibold ${
                          tx.status === 'VERIFIED'
                            ? 'bg-[#00D26A]/10 text-[#00D26A]'
                            : tx.status === 'CLAIMED'
                            ? 'bg-blue-500/10 text-blue-400'
                            : 'bg-rose-500/10 text-rose-400'
                        }`}
                      >
                        {tx.status}
                      </span>
                    </td>
                    <td className="py-2.5 px-3">
                      {tx.claimed ? (
                        <span className="text-[#00D26A] font-medium font-mono text-[11px]">
                          {tx.referenceId || 'Claimed'}
                        </span>
                      ) : (
                        <span className="text-slate-500 text-[11px]">Unclaimed</span>
                      )}
                    </td>
                    <td className="py-2.5 px-3 text-slate-500 text-[11px]">
                      {new Date(tx.createdAt).toLocaleDateString()} {new Date(tx.createdAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
};
